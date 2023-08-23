package acceptance

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/wallarm/specter/lib/ginkgoutil"
	"github.com/wallarm/specter/lib/tag"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var specterBin string

func TestAcceptanceTests(t *testing.T) {
	ginkgoutil.SetupSuite()
	var args []string
	if tag.Race {
		zap.L().Debug("Building with race detector")
		args = append(args, "-race")
	}
	if tag.Debug {
		zap.L().Debug("Building with debug tag")
		args = append(args, "-tags", "debug")
	}
	var err error
	specterBin, err = gexec.Build("github.com/wallarm/specter", args...)
	if err != nil {
		t.Fatal(err)
	}
	defer gexec.CleanupBuildArtifacts()
	RunSpecs(t, "AcceptanceTests Suite")
}

type TestConfig struct {
	SpecterConfig
	// RawConfig overrides Specter.
	RawConfig  string
	ConfigName string            // Without extension. "load" by default.
	UseJSON    bool              // Using YAML by default.
	CmdArgs    []string          // Nothing by default.
	Files      map[string]string // Extra files to put in dir. Ammo, etc.
}

func NewTestConfig() *TestConfig {
	return &TestConfig{
		SpecterConfig: SpecterConfig{
			Pool: []*InstancePoolConfig{NewInstansePoolConfig()},
		},
		Files: map[string]string{},
	}
}

type SpecterConfig struct {
	Pool             []*InstancePoolConfig `yaml:"pools" json:"pools"`
	LogConfig        `yaml:"log,omitempty" json:"log,omitempty"`
	MonitoringConfig `yaml:"monitoring,omitempty" json:"monitoring,omitempty"`
}

type LogConfig struct {
	Level string `yaml:"level,omitempty" json:"level,omitempty"`
	File  string `yaml:"file,omitempty" json:"file,omitempty"`
}

type MonitoringConfig struct {
	Expvar     *expvarConfig     `yaml:"Expvar"`
	CPUProfile *cpuprofileConfig `yaml:"CPUProfile"`
	MemProfile *memprofileConfig `yaml:"MemProfile"`
}

type expvarConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	Port    int  `yaml:"port" json:"port"`
}

type cpuprofileConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	File    string `yaml:"file" json:"file"`
}

type memprofileConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	File    string `yaml:"file" json:"file"`
}

func (pc *SpecterConfig) Append(ipc *InstancePoolConfig) {
	pc.Pool = append(pc.Pool, ipc)
}

func NewInstansePoolConfig() *InstancePoolConfig {
	return &InstancePoolConfig{
		Provider:        map[string]interface{}{},
		Aggregator:      map[string]interface{}{},
		Gun:             map[string]interface{}{},
		RPSSchedule:     map[string]interface{}{},
		StartupSchedule: map[string]interface{}{},
	}

}

type InstancePoolConfig struct {
	ID              string
	Provider        map[string]interface{} `yaml:"ammo" json:"ammo"`
	Aggregator      map[string]interface{} `yaml:"result" json:"result"`
	Gun             map[string]interface{} `yaml:"gun" json:"gun"`
	RPSPerInstance  bool                   `yaml:"rps-per-instance" json:"rps-per-instance"`
	RPSSchedule     interface{}            `yaml:"rps" json:"rps"`
	StartupSchedule interface{}            `yaml:"startup" json:"startup"`
}

type SpecterTester struct {
	*gexec.Session
	// TestDir is working dir of launched specter.
	// It contains config and ammo files, and will be removed after test execution.
	// All files created during a test should created in this dir.
	TestDir string
	Config  *TestConfig
}

func NewTester(conf *TestConfig) *SpecterTester {
	testDir, err := os.MkdirTemp("", "specter_acceptance_")
	Expect(err).ToNot(HaveOccurred())
	if conf.ConfigName == "" {
		conf.ConfigName = "load"
	}
	extension := "yaml"
	if conf.UseJSON {
		extension = "json"
	}
	var confData []byte

	if conf.RawConfig != "" {
		confData = []byte(conf.RawConfig)
	} else {
		if conf.UseJSON {
			confData, err = json.Marshal(conf.SpecterConfig)
		} else {
			confData, err = yaml.Marshal(conf.SpecterConfig)
		}
		Expect(err).ToNot(HaveOccurred())
	}
	confAbsName := filepath.Join(testDir, conf.ConfigName+"."+extension)
	err = os.WriteFile(confAbsName, confData, 0644)
	Expect(err).ToNot(HaveOccurred())

	for file, data := range conf.Files {
		fileAbsName := filepath.Join(testDir, file)
		err = os.WriteFile(fileAbsName, []byte(data), 0644)
		Expect(err).ToNot(HaveOccurred())
	}

	command := exec.Command(specterBin, conf.CmdArgs...)
	command.Dir = testDir
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())

	tt := &SpecterTester{
		Session: session,
		TestDir: testDir,
		Config:  conf,
	}
	return tt
}

func (pt *SpecterTester) ShouldSay(pattern string) {
	EventuallyWithOffset(1, pt.Out, 3*time.Second).Should(gbytes.Say(pattern))
}

func (pt *SpecterTester) ExitCode() int {
	return pt.Session.Wait(5).ExitCode()
}

func (pt *SpecterTester) Close() {
	pt.Terminate()
	_ = os.RemoveAll(pt.TestDir)
}
