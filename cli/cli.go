package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gobuffalo/envy"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/wallarm/specter/core/config"
	"github.com/wallarm/specter/core/engine"
	"github.com/wallarm/specter/helpers"
	"github.com/wallarm/specter/lib/zaputil"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const VersionPandora = "0.5.8"
const Version = "0.0.18"
const defaultConfigFile = "load"
const stdinConfigSelector = "-"
const mainBucket = "wallarm-perf-pandora"

var ConfigSearchDirs = []string{"./", "./config", "/etc/specter", "./../suite/mirroring", "./bin"}

type cliConfig struct {
	Engine     engine.Config    `config:",squash"`
	Log        logConfig        `config:"log"`
	Monitoring monitoringConfig `config:"monitoring"`
}

type logConfig struct {
	Level zapcore.Level `config:"level"`
	File  string        `config:"file"`
}

// TODO: log sampling with WARN when first message is dropped, and WARN at finish with all
// filtered out entries num. Message is filtered out when zapcore.CoreEnable returns true but
// zapcore.Core.Check return nil.
func newLogger(conf logConfig) *zap.Logger {
	zapConf := zap.NewDevelopmentConfig()
	zapConf.OutputPaths = []string{conf.File}
	zapConf.Level.SetLevel(conf.Level)
	logger, err := zapConf.Build(zap.AddCaller())
	if err != nil {
		zap.L().Fatal("Logger build failed", zap.Error(err))
	}
	return logger
}

func defaultConfig() *cliConfig {
	return &cliConfig{
		Log: logConfig{
			Level: zap.InfoLevel,
			File:  "stdout",
		},
		Monitoring: monitoringConfig{
			Expvar: &expvarConfig{
				Enabled: false,
				Port:    1234,
			},
			CPUProfile: &cpuprofileConfig{
				Enabled: false,
				File:    "cpuprofile.log",
			},
			MemProfile: &memprofileConfig{
				Enabled: false,
				File:    "memprofile.log",
			},
		},
	}
}

// TODO: make nice spf13/cobra CLI and integrate it with viper
// TODO: on special command (help or smth else) print list of available plugins

func Run() {
	flag.Usage = func() {
		_, err := fmt.Fprintf(
			os.Stderr,
			"Usage of Specter: specter [<config_filename>]\n"+"<config_filename> is './%s.(yaml|json|...)' by default\n",
			defaultConfigFile)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		flag.PrintDefaults()
	}

	var example, download, upload, update, artefacts bool
	var updateTarget string

	flag.BoolVar(&example, "example", false, "print example config to STDOUT and exit")
	flag.BoolVar(&upload, "upload", false, "upload to s3 ammo and config for specter")
	flag.BoolVar(&artefacts, "artefacts", false, "upload to s3 artefacts for specter")
	flag.BoolVar(&download, "download", false, "download from s3 ammo and config for specter")
	flag.BoolVar(&update, "update", false, "update aim for specter")

	flag.StringVar(&updateTarget, "target", "", "specify the target for update")

	flag.Parse()

	if artefacts {
		s3Client := helpers.Initialize()
		fileNames := []string{"http_phout.log", "phout.log", "answ.log", "load.yaml", "ammo.json"}
		if envy.Get("SPECTER_SEND_SLACK_REPORT", "false") == "true" {
			// TODO: Get picture from Grafana
			// TODO: Upload image to S3

			stressVersions, err := envy.MustGet("OVERLOAD_STRESS_VERSIONS")
			if err != nil {
				logrus.Fatalf("Error getting OVERLOAD_STRESS_VERSIONS: %s", err)
			}

			versions := strings.Split(stressVersions, ";")
			helpers.SendReport(envy.Get("SPECTER_SLACK_WEBHOOK", "none"), helpers.Message{
				Username:     envy.Get("GITLAB_USER_LOGIN", "none"),
				ImageURL:     "", // TODO
				GrafanaURL:   helpers.GenerateGrafanaLink(versions),
				BranchName:   envy.Get("CI_COMMIT_REF_NAME", "none"),
				DeployType:   envy.Get("DEPLOY_TYPE", "none"),
				PipelineLink: envy.Get("CI_PIPELINE_URL", "none"),
				Versions:     versions,
			})
		}

		if err := uploadReportsFiles(s3Client, mainBucket, fileNames...); err != nil {
			logrus.Fatalf("%v", err)
		}

		return
	}

	if upload {
		s3Client := helpers.Initialize()

		fileNames := []string{"load.yaml", "ammo.json"}
		if err := uploadReportsFiles(s3Client, mainBucket, fileNames...); err != nil {
			logrus.Fatalf("%v", err)
		}

		return
	}

	if update {
		if updateTarget == "" {
			logrus.Fatalf("No update target specified")
		}
		if err := helpers.UpdateFiles(updateTarget); err != nil {
			logrus.Fatalf("Error updating files: %s", err)
		}
		return

	}

	if example {
		panic("Not implemented yet")
		// TODO: print example config file content
	}
	logrus.Infof("pandora version : %s", VersionPandora)
	logrus.Infof("specter version : %s", Version)

	ReadConfigAndRunEngine()
}

func ReadConfigAndRunEngine() {
	conf := readConfig(flag.Args())
	RunEngine(conf)
}

func RunEngine(cfg *cliConfig) {

	logger := newLogger(cfg.Log)
	zap.ReplaceGlobals(logger)
	zap.RedirectStdLog(logger)

	closeMonitoring := startMonitoring(cfg.Monitoring)
	defer closeMonitoring()

	m := newEngineMetrics()
	startReport(m)

	specter := engine.New(logger, m, cfg.Engine)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errs := make(chan error)
	go runEngine(ctx, specter, errs)

	// waiting for signal or error message from engine
	termination(specter, cancel, errs, logger)
	logger.Info("Engine run successfully finished")
}

// helper function that awaits specter run
func termination(specter *engine.Engine, gracefulShutdown func(), errs chan error, log *zap.Logger) {
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigs:
		var interruptTimeout = 3 * time.Second
		switch sig {
		case syscall.SIGINT:
			// await gun timeout but no longer than 30 sec.
			interruptTimeout = 30 * time.Second
			log.Info("SIGINT received. Graceful shutdown.", zap.Duration("timeout", interruptTimeout))
			gracefulShutdown()
		case syscall.SIGTERM:
			log.Info("SIGTERM received. Trying to stop gracefully.", zap.Duration("timeout", interruptTimeout))
			gracefulShutdown()
		default:
			log.Fatal("Unexpected signal received. Quiting.", zap.Stringer("signal", sig))
		}

		select {
		case <-time.After(interruptTimeout):
			log.Fatal("Interrupt timeout exceeded")
		case sig := <-sigs:
			log.Fatal("Another signal received. Quiting.", zap.Stringer("signal", sig))
		case err := <-errs:
			log.Fatal("Engine interrupted", zap.Error(err))
		}

	case err := <-errs:
		switch err {
		case nil:
			log.Info("Specter engine successfully finished it's work")
		case err:
			const awaitTimeout = 3 * time.Second
			log.Error("Engine run failed. Awaiting started tasks.", zap.Error(err), zap.Duration("timeout", awaitTimeout))
			gracefulShutdown()
			time.AfterFunc(awaitTimeout, func() {
				log.Fatal("Engine tasks timeout exceeded.")
			})
			specter.Wait()
			log.Fatal("Engine run failed. Specter graceful shutdown successfully finished")
		}
	}
}

func runEngine(ctx context.Context, engine *engine.Engine, errs chan error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errs <- engine.Run(ctx)
}

func readConfig(args []string) *cliConfig {

	var (
		err            error
		logger         *zap.Logger
		useStdinConfig bool
		configBuffer   []byte
	)

	logger, err = zap.NewDevelopment(zap.AddCaller())
	if err != nil {
		panic(err)
	}

	logger = logger.WithOptions(zap.WrapCore(zaputil.NewStackExtractCore))
	zap.ReplaceGlobals(logger)
	zap.RedirectStdLog(logger)

	v := newViper()

	if len(args) > 0 {
		switch {
		case len(args) > 1:
			zap.L().Fatal("Too many command line arguments", zap.Strings("args", args))
		case args[0] == stdinConfigSelector:
			logger.Info("Reading config from standard input")
			useStdinConfig = true
		default:
			v.SetConfigFile(args[0])
		}
	}

	if useStdinConfig {
		v.SetConfigType("yaml")
		configBuffer, err = io.ReadAll(bufio.NewReader(os.Stdin))
		if err != nil {
			logger.Fatal("Cannot read from standard input", zap.Error(err))
		}
		err = v.ReadConfig(strings.NewReader(string(configBuffer)))
		if err != nil {
			logger.Fatal("Config parsing failed", zap.Error(err))
		}

	} else {
		err = v.ReadInConfig()
		logger.Info("Reading config", zap.String("file", v.ConfigFileUsed()))
		if err != nil {
			logger.Fatal("Config read failed", zap.Error(err))
		}
	}

	cfg := defaultConfig()
	err = config.DecodeAndValidate(v.AllSettings(), cfg)
	if err != nil {
		logger.Fatal("Config decode failed", zap.Error(err))
	}
	return cfg
}

func newViper() *viper.Viper {
	v := viper.New()
	v.SetConfigName(defaultConfigFile)
	for _, dir := range ConfigSearchDirs {
		v.AddConfigPath(dir)
	}
	return v
}

type monitoringConfig struct {
	Expvar     *expvarConfig
	CPUProfile *cpuprofileConfig
	MemProfile *memprofileConfig
}

type expvarConfig struct {
	Enabled bool `config:"enabled"`
	Port    int  `config:"port" validate:"required"`
}

type cpuprofileConfig struct {
	Enabled bool   `config:"enabled"`
	File    string `config:"file"`
}

type memprofileConfig struct {
	Enabled bool   `config:"enabled"`
	File    string `config:"file"`
}

func startMonitoring(conf monitoringConfig) (stop func()) {
	zap.L().Debug("Start monitoring", zap.Reflect("conf", conf))
	if conf.Expvar != nil {
		if conf.Expvar.Enabled {
			go func() {
				err := http.ListenAndServe(":"+strconv.Itoa(conf.Expvar.Port), nil)
				zap.L().Fatal("Monitoring server failed", zap.Error(err))
			}()
		}
	}
	var stops []func()
	if conf.CPUProfile.Enabled {
		f, err := os.Create(conf.CPUProfile.File)
		if err != nil {
			zap.L().Fatal("CPU profile file create fail", zap.Error(err))
		}
		zap.L().Info("Starting CPU profiling")
		err = pprof.StartCPUProfile(f)
		if err != nil {
			zap.L().Info("CPU profiling is already enabled")
		}
		stops = append(stops, func() {
			pprof.StopCPUProfile()
			err := f.Close()
			if err != nil {
				zap.L().Info("Error closing CPUProfile file")
			}
		})
	}
	if conf.MemProfile.Enabled {
		f, err := os.Create(conf.MemProfile.File)
		if err != nil {
			zap.L().Fatal("Memory profile file create fail", zap.Error(err))
		}
		stops = append(stops, func() {
			zap.L().Info("Writing memory profile")
			runtime.GC()
			err := pprof.WriteHeapProfile(f)
			if err != nil {
				zap.L().Info("Error writing HeapProfile file")
			}
			err = f.Close()
			if err != nil {
				zap.L().Info("Error closing HeapProfile file")
			}
		})
	}
	stop = func() {
		for _, s := range stops {
			s()
		}
	}
	return
}

func uploadReportsFiles(s3Client *helpers.Client, bucket string, fileNames ...string) error {
	s3Ctx := context.Background()
	for _, name := range fileNames {
		artefactFile, err := helpers.FindFile(name)
		if err != nil {
			logrus.Warnf("error finding %s: %v\n", name, err)
			continue
		}

		if err = helpers.UploadFileToS3(s3Ctx, s3Client, artefactFile, bucket); err != nil {
			return fmt.Errorf("error uploading %s: %w", artefactFile, err)
		}
	}

	return nil
}
