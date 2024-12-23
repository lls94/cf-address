package utils

import (
	"encoding/csv"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultOutput         = "./dist/result.csv"
	maxDelay              = 9999 * time.Millisecond
	minDelay              = 0 * time.Millisecond
	maxLossRate   float32 = 1.0
)

var (
	InputMaxDelay    = maxDelay
	InputMinDelay    = minDelay
	InputMaxLossRate = maxLossRate
	Output           = defaultOutput
	PrintNum         = 10
	coloLimit        = 8
)

// 是否打印测试结果
func NoPrintResult() bool {
	return PrintNum == 0
}

// 是否输出到文件
func noOutput() bool {
	return Output == "" || Output == " "
}

type PingData struct {
	IP       *net.IPAddr
	Sended   int
	Received int
	Delay    time.Duration
	Colo     string
}

type CloudflareIPData struct {
	*PingData
	lossRate      float32
	DownloadSpeed float64
}

// 计算丢包率
func (cf *CloudflareIPData) getLossRate() float32 {
	if cf.lossRate == 0 {
		pingLost := cf.Sended - cf.Received
		cf.lossRate = float32(pingLost) / float32(cf.Sended)
	}
	return cf.lossRate
}

func (cf *CloudflareIPData) toString() []string {
	result := make([]string, 7)
	result[0] = cf.IP.String() + "#" + cf.Colo
	result[1] = strconv.Itoa(cf.Sended)
	result[2] = strconv.Itoa(cf.Received)
	result[3] = strconv.FormatFloat(float64(cf.getLossRate()), 'f', 2, 32)
	result[4] = strconv.FormatFloat(cf.Delay.Seconds()*1000, 'f', 2, 32)
	result[5] = strconv.FormatFloat(cf.DownloadSpeed/1024/1024, 'f', 2, 32)
	result[6] = cf.Colo
	return result
}

// coloMap 机场名称映射
func (cf *CloudflareIPData) toAirport() string {
	airportMap := map[string]string{
		"YUL": "蒙特利尔",
		"LAX": "洛杉矶",
		"PEK": "北京首都",
		"HKG": "香港",
		"SJC": "圣何塞市",
		"IAD": "华盛顿",
		"NRT": "东京",
		"ORD": "芝加哥",
		"JFK": "纽约",
		"CDG": "巴黎",
		"LHR": "伦敦",
		"SIN": "新加坡",
		"FRA": "法兰克福",
		"DXB": "迪拜",
		"SYD": "悉尼",
		"AMS": "阿姆斯特丹",
		"KIX": "大阪",
		"ICN": "首尔",
		"BKK": "曼谷",
		"IST": "伊斯坦",
		"SEA": "西雅图",
		"MUC": "慕尼黑",
		"ZRH": "苏黎世",
		"MAD": "马德里",
		"SFO": "旧金山",
		"YVR": "温哥华",
		"BOM": "孟买",
		"DEL": "新德里",
		"PVG": "上海",
		"TPE": "台北",
		"YYZ": "多伦多",
		"CGK": "雅加达",
		"AKL": "奥克兰",
		"KUL": "吉隆坡",
		"CPT": "开普敦",
		"PEM": "普吉岛",
	}
	if airport, exists := airportMap[cf.Colo]; exists {
		return airport
	}
	return cf.Colo
}

func ExportCsv(data []CloudflareIPData) {
	if noOutput() || len(data) == 0 {
		return
	}

	// 按照 Colo 分组
	colos := make(map[string][]CloudflareIPData)
	for _, v := range data {
		colos[v.Colo] = append(colos[v.Colo], v)
	}

	// 只保留每个 Colo 的前 coloLimit 个 IP
	var limitedData []CloudflareIPData
	for _, ipDataList := range colos {
		// 取每个 colo 的前 coloLimit 个 IP
		count := 0
		for _, ipData := range ipDataList {
			if count >= coloLimit {
				break
			}
			limitedData = append(limitedData, ipData)
			count++
		}
	}

	// 导出到 CSV
	fp, err := os.Create(Output)
	if err != nil {
		log.Fatalf("创建文件[%s]失败：%v", Output, err)
		return
	}
	defer fp.Close()
	w := csv.NewWriter(fp) // 创建一个新的写入文件流
	_ = w.Write([]string{"IP 地址", "已发送", "已接收", "丢包率", "平均延迟", "下载速度 (MB/s)", "数据中心"})
	_ = w.WriteAll(convertToString(limitedData))
	w.Flush()

	// 导出到 TXT
	TxtOutput := strings.TrimSuffix(Output, ".csv") + ".txt"
	txtfp, err := os.Create(TxtOutput)
	if err != nil {
		log.Fatalf("创建文件[%s]失败：%v", TxtOutput, err)
		return
	}
	defer txtfp.Close()
	txtw := csv.NewWriter(txtfp) // 创建一个新的写入文件流
	_ = txtw.WriteAll(convertToStringOnlyIp(limitedData))
	txtw.Flush()
}

func convertToStringOnlyIp(data []CloudflareIPData) [][]string {
	result := make([][]string, 0)
	for _, v := range data {
		// 拼接 IP 和 Colo 字段
		result = append(result, []string{v.IP.String() + "#👍" + v.toAirport()})
	}
	return result
}

func convertToString(data []CloudflareIPData) [][]string {
	result := make([][]string, 0)
	for _, v := range data {
		result = append(result, v.toString())
	}
	return result
}

// 延迟丢包排序
type PingDelaySet []CloudflareIPData

// 延迟条件过滤
func (s PingDelaySet) FilterDelay() (data PingDelaySet) {
	if InputMaxDelay > maxDelay || InputMinDelay < minDelay { // 当输入的延迟条件不在默认范围内时，不进行过滤
		return s
	}
	if InputMaxDelay == maxDelay && InputMinDelay == minDelay { // 当输入的延迟条件为默认值时，不进行过滤
		return s
	}
	for _, v := range s {
		if v.Delay > InputMaxDelay { // 平均延迟上限，延迟大于条件最大值时，后面的数据都不满足条件，直接跳出循环
			break
		}
		if v.Delay < InputMinDelay { // 平均延迟下限，延迟小于条件最小值时，不满足条件，跳过
			continue
		}
		data = append(data, v) // 延迟满足条件时，添加到新数组中
	}
	return
}

// 丢包条件过滤
func (s PingDelaySet) FilterLossRate() (data PingDelaySet) {
	if InputMaxLossRate >= maxLossRate { // 当输入的丢包条件为默认值时，不进行过滤
		return s
	}
	for _, v := range s {
		if v.getLossRate() > InputMaxLossRate { // 丢包几率上限
			break
		}
		data = append(data, v) // 丢包率满足条件时，添加到新数组中
	}
	return
}

func (s PingDelaySet) Len() int {
	return len(s)
}
func (s PingDelaySet) Less(i, j int) bool {
	iRate, jRate := s[i].getLossRate(), s[j].getLossRate()
	if iRate != jRate {
		return iRate < jRate
	}
	return s[i].Delay < s[j].Delay
}
func (s PingDelaySet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// 下载速度排序
type DownloadSpeedSet []CloudflareIPData

func (s DownloadSpeedSet) Len() int {
	return len(s)
}
func (s DownloadSpeedSet) Less(i, j int) bool {
	return s[i].DownloadSpeed > s[j].DownloadSpeed
}
func (s DownloadSpeedSet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s DownloadSpeedSet) Print() {
	if NoPrintResult() {
		return
	}
	if len(s) <= 0 { // IP数组长度(IP数量) 大于 0 时继续
		fmt.Println("\n[信息] 完整测速结果 IP 数量为 0，跳过输出结果。")
		return
	}
	dateString := convertToString(s) // 转为多维数组 [][]String
	if len(dateString) < PrintNum {  // 如果IP数组长度(IP数量) 小于  打印次数，则次数改为IP数量
		PrintNum = len(dateString)
	}
	headFormat := "%-16s%-5s%-5s%-5s%-6s%-11s%-10s\n"
	dataFormat := "%-18s%-8s%-8s%-8s%-10s%-15s%-6s\n"
	for i := 0; i < PrintNum; i++ { // 如果要输出的 IP 中包含 IPv6，那么就需要调整一下间隔
		if len(dateString[i][0]) > 15 {
			headFormat = "%-40s%-5s%-5s%-5s%-6s%-11s%-10s\n"
			dataFormat = "%-42s%-8s%-8s%-8s%-10s%-15s%-6s\n"
			break
		}
	}
	fmt.Printf(headFormat, "IP 地址", "已发送", "已接收", "丢包率", "平均延迟", "下载速度 (MB/s)", "数据中心")
	for i := 0; i < PrintNum; i++ {
		fmt.Printf(dataFormat, dateString[i][0], dateString[i][1], dateString[i][2], dateString[i][3], dateString[i][4], dateString[i][5], dateString[i][6])
	}
	if !noOutput() {
		fmt.Printf("\n完整测速结果已写入 %v 文件，可使用记事本/表格软件查看。\n", Output)
	}
}
