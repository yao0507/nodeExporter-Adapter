package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// 全局变量用于存储标签和互斥锁
var (
	labels     map[string]string
	labelsLock sync.RWMutex
)

// 从文件加载标签
func loadLabels(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var rawLabels map[string]interface{}
	if err := json.Unmarshal(byteValue, &rawLabels); err != nil {
		return nil, err
	}

	newLabels := make(map[string]string)
	for key, value := range rawLabels {
		switch v := value.(type) {
		case string:
			newLabels[key] = v
		case float64:
			newLabels[key] = fmt.Sprintf("%v", v)
		default:
			return nil, fmt.Errorf("unsupported value type for key %s", key)
		}
	}

	return newLabels, nil
}

// 将标签转换为字符串
func mapToLabelString(labels map[string]string) string {
	var sb strings.Builder
	for key, value := range labels {
		sb.WriteString(key + `="` + value + `",`)
	}
	return strings.TrimRight(sb.String(), ",")
}

// 插入新标签到现有标签
func insertLabels(line string, newLabels string) string {
	re := regexp.MustCompile(`(\{.*?\})`)
	matches := re.FindStringSubmatch(line)

	if len(matches) > 0 {
		// 存在标签的情况
		existingLabels := matches[0]
		updatedLabels := strings.TrimRight(existingLabels, "}") + "," + newLabels + "}"
		return strings.Replace(line, existingLabels, updatedLabels, 1)
	} else {
		// 没有标签的情况
		parts := strings.Fields(line)
		if len(parts) > 1 {
			// 在指标名后面插入标签
			return parts[0] + "{" + newLabels + "} " + strings.Join(parts[1:], " ")
		}
		return line
	}
}

// 监控配置文件变化
func watchConfig(filePath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	err = watcher.Add(filePath)
	if err != nil {
		log.Fatalf("Failed to add file to watcher: %v", err)
	}

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("Config file changed, reloading...")
				newLabels, err := loadLabels(filePath)
				if err != nil {
					log.Printf("Error reloading labels: %v", err)
				} else {
					labelsLock.Lock()
					labels = newLabels
					labelsLock.Unlock()
					log.Println("Labels reloaded successfully")
				}
			}
		case err := <-watcher.Errors:
			log.Printf("Watcher error: %v", err)
		}
	}
}

func main() {
	// 使用 pflag 解析命令行参数
	configFilePath := pflag.String("label-config", "", "Path to the labels configuration file (required)")
	port := pflag.Int("port", 9001, "Port to run the adapter on (default 9001)")
	exportURL := pflag.String("export-url", "127.0.0.1:9100/metrics", "URL of the node_exporter (default 127.0.0.1:9100/metrics)")
	pflag.Parse()

	// 检查 label-config 参数是否为空
	if *configFilePath == "" {
		log.Fatal("The --label-config parameter is required and cannot be empty.")
	}

	// 加载初始标签
	var err error
	labels, err = loadLabels(*configFilePath)
	if err != nil {
		log.Fatalf("Error loading labels: %v", err)
	}

	// 启动配置文件监控
	go watchConfig(*configFilePath)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		resp, err := client.Get("http://" + *exportURL)
		if err != nil {
			http.Error(w, "Error contacting node_exporter", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		labelsLock.RLock()
		newLabelsStr := mapToLabelString(labels)
		labelsLock.RUnlock()

		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "#") && line != "" {
				line = insertLabels(line, newLabelsStr)
			}
			w.Write([]byte(line + "\n"))
		}

		if err := scanner.Err(); err != nil {
			http.Error(w, "Error reading response", http.StatusInternalServerError)
		}
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting adapter on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
