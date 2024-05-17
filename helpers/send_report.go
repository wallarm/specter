package helpers

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	"text/template"
)

type Message struct {
	Username     string
	ImageURL     string
	GrafanaURL   string
	DeployType   string
	BranchName   string
	PipelineLink string
	Versions     []string
}

func SendReport(targetURL string, message Message) {
	logrus.Printf("Deploy Type: %s", message.DeployType)

	tmpl, err := getTemplate() // Get the template based on the deployment type
	if err != nil {
		logrus.Fatalf("Error occurred while getting template: %s", err)
	}

	tpl := new(bytes.Buffer)
	if err = tmpl.Execute(tpl, message); err != nil {
		logrus.Fatalf("Error occurred while executing template: %s", err)
	}

	logrus.Printf("Template: %s", tpl.String())

	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewBuffer(tpl.Bytes()))
	if err != nil {
		log.Fatalf("Error occurred while creating HTTP request: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error occurred while sending HTTP request: %s", err)
	}
	respBody := new(bytes.Buffer)
	respBody.ReadFrom(resp.Body)
	logrus.Printf("Response Body: %s", respBody.String())
	defer resp.Body.Close()
	logrus.Println(fmt.Sprintf("Response Status: %s", resp.Status))
}

func getTemplate() (*template.Template, error) {
	// TODO: screenshot
	//            {
	//                "type": "image",
	//                "image_url": "{{.ImageURL}}",
	//                "alt_text": "Performance Screenshot"
	//            },
	baseText := `{
        "blocks": [
            {"type": "section", "text": {"type": "mrkdwn", "text": "Hello, <@{{.Username}}>! Here is{{if ne .DeployType "none"}} the {{.DeployType}}{{end}} performance update:"}},
            {{if ne .DeployType "none"}}
            {"type": "section", "text": {"type": "mrkdwn", "text": "Check out the detailed performance metrics on Grafana: <{{.GrafanaURL}}|Grafana Dashboard>"}},
            {{end}}
            {"type": "section", "text": {"type": "mrkdwn", "text": " <{{.PipelineLink}}|Pipeline> triggered on branch: *{{.BranchName}}*"}}
        ]
    }`
	return template.New("slackMessage").Parse(baseText)
}
