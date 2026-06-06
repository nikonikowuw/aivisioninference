// Package fingerprint 提供多级瀑布式硬件指纹提取引擎。
// 依优先级采集 NPU/CPU Serial、主板 UUID 及 eMMC/磁盘序列号，
// 生成唯一的设备指纹用于绑定授权，彻底排除网卡相关标识。
package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

// Salt 加盐字符串，与硬件标识组合后进行 Hash，防止指纹被轻易猜出
const Salt = "AIVisionInference_Fingerprint_Salt_v1"

// 硬件标识优先级（Tier 1 -> Tier 2 -> Tier 3）
// Tier 1: NPU S/N、CPU Serial（核心计算单元）
// Tier 2: 主板 UUID（主板与固件）
// Tier 3: eMMC CID、系统盘 Serial（核心存储介质）

// hwIdentifier 代表一个探测到的硬件标识
type hwIdentifier struct {
	key   string // 唯一标识键名
	value string // 标识值
	tier  int    // 优先级层级（1=最高）
}

// Extract 提取硬件指纹。
// 按优先级探测 NPU/CPU Serial、主板 UUID、eMMC/磁盘序列号，
// 将可用标识组合后加盐 SHA-256，返回 hex 编码的指纹字符串。
// 如果任何探测方式返回错误，静默跳过（降级处理），不中断流程。
func Extract() (string, error) {
	identifiers := probeAll()
	if len(identifiers) == 0 {
		// 无法提取任何硬件标识，使用设备名作为最终兜底
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "unknown-device"
		}
		identifiers = append(identifiers, hwIdentifier{key: "hostname", value: hostname, tier: 99})
	}

	// 按 tier 排序，保证不同运行顺序产生相同结果
	sort.Slice(identifiers, func(i, j int) bool {
		if identifiers[i].tier != identifiers[j].tier {
			return identifiers[i].tier < identifiers[j].tier
		}
		return identifiers[i].key < identifiers[j].key
	})

	// 拼接：key:value 用 | 分隔
	var parts []string
	for _, id := range identifiers {
		parts = append(parts, id.key+":"+id.value)
	}
	combined := strings.Join(parts, "|")

	// 加盐 SHA-256
	h := sha256.New()
	h.Write([]byte(combined + "|" + Salt))
	return hex.EncodeToString(h.Sum(nil))[:32], nil // 截取前 32 位（128 bit）
}

// probeAll 调用所有探测函数，收集可用的硬件标识
func probeAll() []hwIdentifier {
	var ids []hwIdentifier

	// Tier 1: NPU S/N (华为昇腾 npu-smi)
	if v := probeNPU(); v != "" {
		ids = append(ids, hwIdentifier{key: "npu_sn", value: v, tier: 1})
	}

	// Tier 1: CPU Serial (/proc/cpuinfo)
	if v := probeCPUSerial(); v != "" {
		ids = append(ids, hwIdentifier{key: "cpu_serial", value: v, tier: 1})
	}

	// Tier 2: 主板 UUID (dmidecode 或 /sys/class/dmi)
	if v := probeBoardUUID(); v != "" {
		ids = append(ids, hwIdentifier{key: "board_uuid", value: v, tier: 2})
	}

	// Tier 3: eMMC CID (/sys/class/block/mmcblk0/device/cid)
	if v := probeEMMCCID(); v != "" {
		ids = append(ids, hwIdentifier{key: "emmc_cid", value: v, tier: 3})
	}

	// Tier 3: 系统盘 Serial (lsblk)
	if v := probeDiskSerial(); v != "" {
		ids = append(ids, hwIdentifier{key: "disk_serial", value: v, tier: 3})
	}

	return ids
}

// probeNPU 尝试通过 npu-smi 获取华为昇腾 NPU 序列号
func probeNPU() string {
	out, err := exec.Command("npu-smi", "info").Output()
	if err != nil {
		return ""
	}
	// 匹配 "Serial Number: XXXXX" 或类似格式
	re := regexp.MustCompile(`(?i)serial\s*(number|no|id)?\s*[:=]\s*(\S+)`)
	matches := re.FindStringSubmatch(string(out))
	if len(matches) >= 3 {
		return strings.TrimSpace(matches[2])
	}
	return ""
}

// probeCPUSerial 从 /proc/cpuinfo 提取 CPU Serial（ARM 设备通常有此字段）
func probeCPUSerial() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`(?m)^Serial\s*:\s*(\S+)`)
	matches := re.FindStringSubmatch(string(data))
	if len(matches) >= 2 {
		v := strings.TrimSpace(matches[1])
		// 某些 x86 设备返回全零，忽略
		if v == "" || v == "0000000000000000" {
			return ""
		}
		return v
	}
	return ""
}

// probeBoardUUID 尝试从 /sys/class/dmi/id/product_uuid 获取主板 UUID
func probeBoardUUID() string {
	// 优先 sysfs
	data, err := os.ReadFile("/sys/class/dmi/id/product_uuid")
	if err == nil {
		v := strings.TrimSpace(string(data))
		if v != "" && v != "00000000-0000-0000-0000-000000000000" {
			return v
		}
	}
	// 回退到 dmidecode（需要 root 权限）
	out, err := exec.Command("dmidecode", "-s", "system-uuid").Output()
	if err != nil {
		return ""
	}
	v := strings.TrimSpace(string(out))
	if v != "" && v != "00000000-0000-0000-0000-000000000000" {
		return v
	}
	return ""
}

// probeEMMCCID 从 /sys/class/block/mmcblk0/device/cid 获取 eMMC CID（嵌入式 ARM 设备常用）
func probeEMMCCID() string {
	data, err := os.ReadFile("/sys/class/block/mmcblk0/device/cid")
	if err != nil {
		return ""
	}
	v := strings.TrimSpace(string(data))
	if v != "" {
		return v
	}
	return ""
}

// probeDiskSerial 通过 lsblk 获取系统盘的序列号
func probeDiskSerial() string {
	// 找到根分区对应的设备
	rootDev := findRootDevice()
	if rootDev == "" {
		return ""
	}
	// lsblk -nod -o serial /dev/sda
	out, err := exec.Command("lsblk", "-nod", "-o", "serial", rootDev).Output()
	if err != nil {
		return ""
	}
	v := strings.TrimSpace(string(out))
	if v != "" {
		return v
	}
	return ""
}

// findRootDevice 查找根分区挂载的块设备（如 /dev/sda、/dev/mmcblk0）
func findRootDevice() string {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == "/" {
			dev := parts[0]
			// 去掉分区号，获取整盘设备名（如 /dev/sda1 -> /dev/sda, /dev/mmcblk0p1 -> /dev/mmcblk0）
			re := regexp.MustCompile(`^(/dev/(?:sd[a-z]|mmcblk\d+|nvme\d+n\d+))`)
			m := re.FindStringSubmatch(dev)
			if len(m) >= 2 {
				return m[1]
			}
		}
	}
	return ""
}
