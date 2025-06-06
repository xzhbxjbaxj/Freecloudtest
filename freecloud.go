package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

// 配置常量
const (
	LOGIN_URL   = "https://freecloud.ltd/login"
	CONSOLE_URL = "https://freecloud.ltd/member/index"
	RENEW_URL   = "https://freecloud.ltd/server/detail/%d/renew"
)

// 响应结构体
type Response struct {
	Msg string `json:"msg"`
}

type FCProfile struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Machines []int  `json:"machines"`
}

func init() {
	// 配置日志格式
	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)
	log.SetPrefix("[INFO] ")
}

func main() {
	// 解析命令行参数
	configFlag := flag.String("c", "", "Single FCProfile in JSON format")
	flag.Parse()

	var profiles []FCProfile

	// 处理命令行参数
	if *configFlag != "" {
		var profile FCProfile
		if err := json.Unmarshal([]byte(*configFlag), &profile); err != nil {
			log.Printf("❌ Invalid JSON format: %v\n", err)
			os.Exit(1)
		}
		profiles = append(profiles, profile)
	} else {
		// 处理环境变量
		envFCProfiles := os.Getenv("FC_PROFILES")
		if envFCProfiles == "" {
			log.Println("❌ No configuration provided via CLI or FC_PROFILES")
			os.Exit(1)
		}

		// 修复JSON格式兼容性
		envFCProfiles = strings.TrimSpace(envFCProfiles)
		if !strings.HasPrefix(envFCProfiles, "[") {
			envFCProfiles = "[" + envFCProfiles + "]"
		}

		if err := json.Unmarshal([]byte(envFCProfiles), &profiles); err != nil {
			log.Printf("❌ Invalid FC_PROFILES format: %v\n", err)
			os.Exit(1)
		}
	}

	// 执行更新操作
	for _, p := range profiles {
		log.Printf("🔑 Processing user: %s\n", p.Username)
		renew(p.Username, p.Password, p.Machines)
	}
}

func renew(username, password string, machineIDs []int) {
	// 创建带Cookie的HTTP客户端
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal("创建Cookie管理器失败:", err)
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 20 * time.Second,
	}

	if login(client, username, password) {
		for _, machineID := range machineIDs {
			renewServer(client, machineID)
		}
	}
}

// login 模拟登录并返回是否成功
func login(client *http.Client, username, password string) bool {
	log.Println("🚀 正在尝试登录 FreeCloud...")

	// 准备登录表单数据
	formData := url.Values{
		"username":    {username},
		"password":    {password},
		"mobile":      {""},
		"captcha":     {""},
		"verify_code": {""},
		"agree":       {"1"},
		"login_type":  {"PASS"},
		"submit":      {"1"},
	}

	// 创建请求
	req, err := http.NewRequest("POST", LOGIN_URL, strings.NewReader(formData.Encode()))
	if err != nil {
		log.Printf("❌ 创建登录请求失败: %v", err)
		return false
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://freecloud.ltd/login")
	req.Header.Set("Origin", "https://freecloud.ltd")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ 登录请求失败: %v", err)
		return false
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ 读取登录响应失败: %v", err)
		return false
	}

	// 检查登录是否成功
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "退出登录") && !strings.Contains(bodyStr, "member/index") {
		log.Printf("❌ 登录失败，请检查用户名或密码是否正确。")
		return false
	}

	// 访问控制台页面确认登录状态
	_, err = client.Get(CONSOLE_URL)
	if err != nil {
		log.Printf("❌ 访问控制台页面失败: %v", err)
		return false
	}

	log.Println("✅ 登录成功！")
	return true
}

// renewServer 为服务器续费
func renewServer(client *http.Client, machineID int) {
	log.Printf("🔄 正在尝试为服务器 %s 续费...", machineID)

	// 准备续费表单数据
	formData := url.Values{
		"month":     {"1"},
		"submit":    {"1"},
		"coupon_id": {"0"},
	}

	// 创建请求
	renewURL := fmt.Sprintf(RENEW_URL, machineID)
	req, err := http.NewRequest("POST", renewURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		log.Printf("❌ 创建续费请求失败: %v", err)
		return
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://freecloud.ltd/login")
	req.Header.Set("Origin", "https://freecloud.ltd")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ 续费请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ 读取续费响应失败: %v", err)
		return
	}

	// 尝试解析JSON响应
	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		log.Println("⚠️ 返回内容不是 JSON，原始响应如下：")
		log.Println(string(body))
		return
	}

	// 处理续费结果
	message := response.Msg
	if message == "请在到期前3天后再续费" {
		log.Printf("⚠️ 续费状态：%s", message)
	} else if message == "续费成功" {
		log.Printf("✅ 续费状态：%s", message)
	} else {
		log.Printf("请检查FC_MACHINE_IDS是否输入正确")
		log.Printf("%s", message)
		os.Exit(1)
	}
}
