// Package service 提供业务逻辑层
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NetworkInterfaceInfo 网络接口信息
type NetworkInterfaceInfo struct {
	Name       string   `json:"name"`
	MAC        string   `json:"mac"`
	State      string   `json:"state"`        // up/down
	ConfigMode string   `json:"config_mode"`  // dhcp/static
	IPAddress  string   `json:"ip_address"`
	CIDR       int      `json:"cidr"`
	Gateway    string   `json:"gateway"`
	DNSServers []string `json:"dns_servers"`
	IsCurrent  bool     `json:"is_current"`
}

// NetworkConfigRequest 网络配置请求
type NetworkConfigRequest struct {
	Interfaces []InterfaceConfigRequest `json:"interfaces" binding:"required"`
}

// InterfaceConfigRequest 单个网卡配置请求
type InterfaceConfigRequest struct {
	Name       string   `json:"name" binding:"required"`
	ConfigMode string   `json:"config_mode" binding:"required,oneof=dhcp static"`
	IPAddress  string   `json:"ip_address,omitempty"`
	CIDR       int      `json:"cidr,omitempty"`
	Gateway    string   `json:"gateway,omitempty"`
	DNSServers []string `json:"dns_servers,omitempty"`
}

// RollbackTransaction 回滚事务
type RollbackTransaction struct {
	TransactionID      string            `json:"transaction_id"`
	CreatedAt          time.Time         `json:"created_at"`
	CurrentAccessIFace string            `json:"current_access_iface"`
	Interfaces         []InterfaceBackup `json:"interfaces"`
}

// InterfaceBackup 网卡备份
type InterfaceBackup struct {
	Name       string                `json:"name"`
	BackupIP   string                `json:"backup_ip"`
	BackupCIDR int                   `json:"backup_cidr"`
	NewConfig  InterfaceConfigRequest `json:"new_config"`
}

// NetworkService 网络配置服务
type NetworkService struct {
	rollbackDir string
}

// NewNetworkService 创建网络配置服务
// db 参数为预留接口，当前不使用数据库，保留签名以便后续扩展
func NewNetworkService(_ *gorm.DB) *NetworkService {
	rollbackDir := "/var/lib/network-rollback"
	os.MkdirAll(rollbackDir, 0700)

	return &NetworkService{
		rollbackDir: rollbackDir,
	}
}

// GetInterfaces 获取所有网络接口
func (s *NetworkService) GetInterfaces() ([]NetworkInterfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %v", err)
	}

	// 获取当前连接网卡
	currentIFace, _ := s.getCurrentAccessInterface()

	var interfaces []NetworkInterfaceInfo
	for _, iface := range ifaces {
		if isVirtualInterface(iface.Name) {
			continue
		}

		// 获取状态
		state := "down"
		if iface.Flags&net.FlagUp != 0 {
			state = "up"
		}

		// 获取 IP 地址
		addrs, err := iface.Addrs()
		ipAddr := ""
		cidr := 0
		if err == nil {
			for _, addr := range addrs {
				if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
					ipAddr = ipNet.IP.String()
					ones, _ := ipNet.Mask.Size()
					cidr = ones
					break
				}
			}
		}

		// 只显示有 IP 的接口
		if ipAddr == "" {
			continue
		}

		// 获取 DNS
		dnsServers := s.getDNSServers()

		// 获取配置模式
		configMode := s.detectConfigMode(iface.Name)

		interfaces = append(interfaces, NetworkInterfaceInfo{
			Name:       iface.Name,
			MAC:        iface.HardwareAddr.String(),
			State:      state,
			ConfigMode: configMode,
			IPAddress:  ipAddr,
			CIDR:       cidr,
			Gateway:    s.getGateway(iface.Name),
			DNSServers: dnsServers,
			IsCurrent:  iface.Name == currentIFace,
		})
	}

	return interfaces, nil
}

// detectConfigMode 检测网卡配置模式（DHCP 或静态）
func (s *NetworkService) detectConfigMode(ifaceName string) string {
	if runtime.GOOS != "linux" {
		return "dhcp"
	}

	// 尝试读取 netplan 配置
	data, err := os.ReadFile(fmt.Sprintf("/etc/netplan/01-%s.yaml", ifaceName))
	if err == nil {
		content := string(data)
		if strings.Contains(content, "dhcp4: true") || strings.Contains(content, "dhcp6: true") {
			return "dhcp"
		}
		if strings.Contains(content, "addresses:") {
			return "static"
		}
	}

	// 尝试读取 interfaces 文件
	data, err = os.ReadFile("/etc/network/interfaces")
	if err == nil {
		content := string(data)
		if strings.Contains(content, ifaceName) && strings.Contains(content, "dhcp") {
			return "dhcp"
		}
	}

	return "dhcp"
}

// getGateway 获取指定网卡的默认网关
func (s *NetworkService) getGateway(ifaceName string) string {
	if runtime.GOOS != "linux" {
		return ""
	}

	cmd := exec.Command("ip", "route", "show", "dev", ifaceName, "default")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		for i, field := range fields {
			if field == "via" && i+1 < len(fields) {
				return fields[i+1]
			}
		}
	}

	return ""
}

// getCurrentAccessInterface 获取当前访问使用的网卡
func (s *NetworkService) getCurrentAccessInterface() (string, error) {
	switch runtime.GOOS {
	case "linux":
		return s.getCurrentAccessInterfaceLinux()
	default:
		return s.getCurrentAccessInterfaceFallback()
	}
}

// getCurrentAccessInterfaceLinux Linux 平台获取当前访问网卡
func (s *NetworkService) getCurrentAccessInterfaceLinux() (string, error) {
	cmd := exec.Command("ip", "route", "show", "default")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "dev") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "dev" && i+1 < len(parts) {
					return parts[i+1], nil
				}
			}
		}
	}

	return "", fmt.Errorf("no default route found")
}

// getCurrentAccessInterfaceFallback 降级方案获取当前访问网卡
func (s *NetworkService) getCurrentAccessInterfaceFallback() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		if iface.Name == "lo" || iface.Name == "lo0" {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
				return iface.Name, nil
			}
		}
	}

	return "", fmt.Errorf("no active interface found")
}

// getDNSServers 获取 DNS 服务器列表
func (s *NetworkService) getDNSServers() []string {
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return []string{}
	}

	var servers []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "nameserver") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				servers = append(servers, fields[1])
			}
		}
	}

	return servers
}

// ApplyConfig 应用网络配置
// 流程: 备份当前配置 → 应用新配置 → 返回事务(含回滚信息)
func (s *NetworkService) ApplyConfig(req *NetworkConfigRequest) (*RollbackTransaction, error) {
	transactionID := generateUUID()

	// 获取当前连接网卡
	currentIFace, _ := s.getCurrentAccessInterface()

	transaction := &RollbackTransaction{
		TransactionID:      transactionID,
		CreatedAt:          time.Now(),
		CurrentAccessIFace: currentIFace,
		Interfaces:         make([]InterfaceBackup, 0, len(req.Interfaces)),
	}

	transactionDir := filepath.Join(s.rollbackDir, fmt.Sprintf("transaction-%s", transactionID))
	if err := os.MkdirAll(transactionDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create transaction dir: %w", err)
	}

	// 备份并应用每个接口配置
	for _, ifaceReq := range req.Interfaces {
		backup, err := s.backupAndApply(transactionID, transactionDir, ifaceReq)
		if err != nil {
			// 应用失败时回滚已应用的接口
			s.rollbackAppliedInterfaces(transaction)
			return nil, fmt.Errorf("failed to apply config for %s: %w", ifaceReq.Name, err)
		}
		transaction.Interfaces = append(transaction.Interfaces, *backup)
	}

	// 持久化事务信息
	transactionData, _ := json.Marshal(transaction)
	if err := os.WriteFile(filepath.Join(transactionDir, "rollback.json"), transactionData, 0600); err != nil {
		zap.L().Warn("failed to persist transaction data", zap.Error(err))
	}

	// 启动回滚守护协程（超时未确认则自动回滚）
	go s.rollbackWatcher(transactionID, 120*time.Second)

	return transaction, nil
}

// backupAndApply 备份当前接口配置并应用新配置
func (s *NetworkService) backupAndApply(transactionID, transactionDir string, req InterfaceConfigRequest) (*InterfaceBackup, error) {
	// 1. 备份当前 IP
	currentIP, currentCIDR := s.getCurrentIP(req.Name)

	backup := &InterfaceBackup{
		Name:       req.Name,
		BackupIP:   currentIP,
		BackupCIDR: currentCIDR,
		NewConfig:  req,
	}

	// 2. 保存备份配置文件（用于回滚）
	backupData, _ := json.Marshal(backup)
	backupFile := filepath.Join(transactionDir, fmt.Sprintf("backup-%s.json", req.Name))
	if err := os.WriteFile(backupFile, backupData, 0600); err != nil {
		return nil, fmt.Errorf("failed to save backup: %w", err)
	}

	// 3. 应用新配置
	if err := s.applyInterfaceConfig(req); err != nil {
		return nil, err
	}

	return backup, nil
}

// getCurrentIP 获取接口当前 IP 和 CIDR
func (s *NetworkService) getCurrentIP(ifaceName string) (string, int) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return "", 0
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", 0
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
			ones, _ := ipNet.Mask.Size()
			return ipNet.IP.String(), ones
		}
	}

	return "", 0
}

// applyInterfaceConfig 应用单个接口的网络配置
func (s *NetworkService) applyInterfaceConfig(req InterfaceConfigRequest) error {
	if runtime.GOOS != "linux" {
		zap.L().Warn("network config apply is only supported on Linux",
			zap.String("interface", req.Name),
			zap.String("os", runtime.GOOS))
		return nil // 非 Linux 平台跳过实际配置，但不阻断流程
	}

	// 生成 netplan 配置
	netplanContent := s.generateNetplanConfig(req)
	netplanPath := fmt.Sprintf("/etc/netplan/01-%s.yaml", req.Name)

	// 写入 netplan 配置
	if err := os.WriteFile(netplanPath, []byte(netplanContent), 0600); err != nil {
		return fmt.Errorf("failed to write netplan config: %w", err)
	}

	// 应用 netplan（不中断网络 —— 使用 --timeout）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "netplan", "apply")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netplan apply failed: %s: %w", string(output), err)
	}

	zap.L().Info("network config applied",
		zap.String("interface", req.Name),
		zap.String("mode", req.ConfigMode),
		zap.String("ip", req.IPAddress))

	return nil
}

// generateNetplanConfig 生成 netplan YAML 配置
func (s *NetworkService) generateNetplanConfig(req InterfaceConfigRequest) string {
	var sb strings.Builder

	sb.WriteString("network:\n")
	sb.WriteString("  version: 2\n")
	sb.WriteString("  ethernets:\n")
	sb.WriteString(fmt.Sprintf("    %s:\n", req.Name))

	if req.ConfigMode == "dhcp" {
		sb.WriteString("      dhcp4: true\n")
	} else {
		sb.WriteString("      dhcp4: false\n")
		if req.IPAddress != "" && req.CIDR > 0 {
			sb.WriteString("      addresses:\n")
			sb.WriteString(fmt.Sprintf("        - %s/%d\n", req.IPAddress, req.CIDR))
		}
		if req.Gateway != "" {
			sb.WriteString("      routes:\n")
			sb.WriteString("        - to: default\n")
			sb.WriteString(fmt.Sprintf("          via: %s\n", req.Gateway))
		}
		if len(req.DNSServers) > 0 {
			sb.WriteString("      nameservers:\n")
			sb.WriteString("        addresses:\n")
			for _, dns := range req.DNSServers {
				sb.WriteString(fmt.Sprintf("          - %s\n", dns))
			}
		}
	}

	return sb.String()
}

// rollbackAppliedInterfaces 回滚已应用的接口配置
func (s *NetworkService) rollbackAppliedInterfaces(transaction *RollbackTransaction) {
	for _, backup := range transaction.Interfaces {
		if err := s.applyRollback(backup); err != nil {
			zap.L().Error("failed to rollback interface",
				zap.String("interface", backup.Name),
				zap.Error(err))
		}
	}
}

// applyRollback 回滚单个接口配置
func (s *NetworkService) applyRollback(backup InterfaceBackup) error {
	if runtime.GOOS != "linux" {
		return nil
	}

	// 生成回滚 netplan 配置
	var sb strings.Builder
	sb.WriteString("network:\n")
	sb.WriteString("  version: 2\n")
	sb.WriteString("  ethernets:\n")
	sb.WriteString(fmt.Sprintf("    %s:\n", backup.Name))

	if backup.BackupIP != "" && backup.BackupCIDR > 0 {
		sb.WriteString("      dhcp4: false\n")
		sb.WriteString("      addresses:\n")
		sb.WriteString(fmt.Sprintf("        - %s/%d\n", backup.BackupIP, backup.BackupCIDR))
	} else {
		sb.WriteString("      dhcp4: true\n")
	}

	netplanPath := fmt.Sprintf("/etc/netplan/01-%s.yaml", backup.Name)
	if err := os.WriteFile(netplanPath, []byte(sb.String()), 0600); err != nil {
		return fmt.Errorf("failed to write rollback netplan: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "netplan", "apply")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netplan apply rollback failed: %s: %w", string(output), err)
	}

	zap.L().Info("interface rolled back", zap.String("interface", backup.Name))
	return nil
}

// rollbackWatcher 回滚守护协程：超时未确认则自动回滚
func (s *NetworkService) rollbackWatcher(transactionID string, timeout time.Duration) {
	transactionDir := filepath.Join(s.rollbackDir, fmt.Sprintf("transaction-%s", transactionID))
	confirmFile := filepath.Join(transactionDir, "confirm")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeoutCh := time.After(timeout)

	for {
		select {
		case <-ticker.C:
			if _, err := os.Stat(confirmFile); err == nil {
				// 已确认，清理事务文件
				zap.L().Info("network config confirmed", zap.String("transaction_id", transactionID))
				return
			}
		case <-timeoutCh:
			// 超时未确认，自动回滚
			zap.L().Warn("network config not confirmed within timeout, rolling back",
				zap.String("transaction_id", transactionID),
				zap.Duration("timeout", timeout))

			rollbackData, err := os.ReadFile(filepath.Join(transactionDir, "rollback.json"))
			if err != nil {
				zap.L().Error("failed to read rollback data", zap.Error(err))
				return
			}

			var transaction RollbackTransaction
			if err := json.Unmarshal(rollbackData, &transaction); err != nil {
				zap.L().Error("failed to parse rollback data", zap.Error(err))
				return
			}

			s.rollbackAppliedInterfaces(&transaction)

			// 清理事务目录
			os.RemoveAll(transactionDir)
			return
		}
	}
}

// ConfirmConfig 确认配置生效
func (s *NetworkService) ConfirmConfig(transactionID string) error {
	transactionDir := filepath.Join(s.rollbackDir, fmt.Sprintf("transaction-%s", transactionID))
	confirmFile := filepath.Join(transactionDir, "confirm")

	// 验证事务存在
	rollbackFile := filepath.Join(transactionDir, "rollback.json")
	if _, err := os.Stat(rollbackFile); os.IsNotExist(err) {
		return fmt.Errorf("transaction not found: %s", transactionID)
	}

	return os.WriteFile(confirmFile, []byte("confirmed"), 0600)
}

// RollbackConfig 手动回滚配置
func (s *NetworkService) RollbackConfig(transactionID string) error {
	transactionDir := filepath.Join(s.rollbackDir, fmt.Sprintf("transaction-%s", transactionID))
	rollbackFile := filepath.Join(transactionDir, "rollback.json")

	data, err := os.ReadFile(rollbackFile)
	if err != nil {
		return fmt.Errorf("transaction not found: %s", transactionID)
	}

	var transaction RollbackTransaction
	if err := json.Unmarshal(data, &transaction); err != nil {
		return fmt.Errorf("invalid transaction data: %w", err)
	}

	// 执行回滚
	s.rollbackAppliedInterfaces(&transaction)

	// 清理事务目录
	os.RemoveAll(transactionDir)

	return nil
}

func generateUUID() string {
	return uuid.New().String()
}

// virtualInterfacePrefixes 需要过滤的虚拟接口前缀列表
var virtualInterfacePrefixes = []string{"lo", "utun", "awdl", "llw", "gif", "stf", "bridge", "vnic", "vmnet", "docker", "br-", "veth", "ap", "anpi"}

func isVirtualInterface(name string) bool {
	for _, prefix := range virtualInterfacePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
