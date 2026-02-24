package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"gemini-web-to-api/internal/config"
	"gemini-web-to-api/internal/providers/gemini"
	"github.com/imroc/req/v3"
	"go.uber.org/zap"
)

func main() {
	cfg := &config.Config{}
	cfg.Gemini.Secure1PSID = os.Getenv("GEMINI_1PSID")
	cfg.Gemini.Secure1PSIDTS = os.Getenv("GEMINI_1PSIDTS")
	cfg.Gemini.Secure1PSIDCC = os.Getenv("GEMINI_1PSIDCC")

	logger, _ := zap.NewDevelopment()
	client := gemini.NewClient(cfg, logger)

	ctx := context.Background()
	if err := client.Init(ctx); err != nil {
		log.Fatalf("Failed to init client: %v", err)
	}

	artifactID := os.Getenv("ARTIFACT_ID")
	if artifactID == "" {
		artifactID = "af68710d-273f-4945-8568-aef04e40a0c3"
	}

	// Build payload matching kwDCne format found in browser
	inner := []interface{}{artifactID}
	innerJSON, _ := json.Marshal(inner)
	fReq := [][][]interface{}{
		{
			{"kwDCne", string(innerJSON), nil, "generic"},
		},
	}
	fReqJSON, _ := json.Marshal(fReq)
	formData := url.Values{
		"f.req": {string(fReqJSON)},
	}

	reqClient := req.NewClient()
	reqClient.SetTimeout(60 * time.Second)
	reqClient.SetCommonHeader("Accept", "*/*")
	reqClient.SetCommonHeader("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	reqClient.SetCommonHeader("Cookie",
		fmt.Sprintf("__Secure-1PSID=%s; __Secure-1PSIDTS=%s; __Secure-1PSIDCC=%s",
			cfg.Gemini.Secure1PSID, cfg.Gemini.Secure1PSIDTS, cfg.Gemini.Secure1PSIDCC))

	resp, err := reqClient.R().
		SetContext(ctx).
		SetFormDataFromValues(formData).
		Post("https://gemini.google.com/_/BardChatUi/data/batchexecute?rpcids=kwDCne&bl=boq_assistant-bard-web-server_20250208.10_p1&f.sid=-3456&bgr=0&hl=en&_reqid=1234&rt=c")
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}

	body := resp.String()
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ")]}'") {
			continue
		}
		var envelope []interface{}
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			continue
		}
		for _, item := range envelope {
			itemArr, ok := item.([]interface{})
			if !ok || len(itemArr) < 3 || itemArr[0] != "wrb.fr" {
				continue
			}
			payloadStr, ok := itemArr[2].(string)
			if !ok {
				continue
			}
			// Write raw payload to file
			os.WriteFile("scripts/raw_payload.json", []byte(payloadStr), 0644)
			fmt.Println("Raw payload written to scripts/raw_payload.json")

			// Print top-level structure
			var payload []interface{}
			json.Unmarshal([]byte(payloadStr), &payload)
			printStructure(payload, 0, 5)
			return
		}
	}
	fmt.Println("No payload found")
}

func printStructure(v interface{}, depth, maxDepth int) {
	if depth > maxDepth {
		fmt.Printf("%s...\n", indent(depth))
		return
	}
	switch val := v.(type) {
	case []interface{}:
		fmt.Printf("%s[array len=%d]\n", indent(depth), len(val))
		for i, item := range val {
			if i > 10 {
				fmt.Printf("%s...(+%d more)\n", indent(depth+1), len(val)-i)
				break
			}
			fmt.Printf("%s[%d]:\n", indent(depth+1), i)
			printStructure(item, depth+2, maxDepth)
		}
	case map[string]interface{}:
		fmt.Printf("%s{map len=%d}\n", indent(depth), len(val))
	case string:
		if len(val) > 120 {
			fmt.Printf("%sstring(%d chars): %s...\n", indent(depth), len(val), val[:120])
		} else {
			fmt.Printf("%sstring: %q\n", indent(depth), val)
		}
	case float64:
		fmt.Printf("%sfloat64: %v\n", indent(depth), val)
	case bool:
		fmt.Printf("%sbool: %v\n", indent(depth), val)
	case nil:
		fmt.Printf("%snil\n", indent(depth))
	default:
		fmt.Printf("%s(%T)\n", indent(depth), val)
	}
}

func indent(n int) string {
	return strings.Repeat("  ", n)
}
