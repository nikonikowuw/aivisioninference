package service

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/niko-admin/niko-admin/internal/model"
)

// ====== MANSCDP XML 数据结构 ======

// ChannelInfo 设备通道信息（解析自 GB28181 目录响应）
type ChannelInfo struct {
	DeviceID     string  `json:"device_id"`
	Name         string  `json:"name"`
	Manufacturer string  `json:"manufacturer"`
	Model        string  `json:"model"`
	Status       string  `json:"status"`
	Owner        string  `json:"owner"`
	CivilCode    string  `json:"civil_code"`
	Address      string  `json:"address"`
	Parental     int     `json:"parental"`
	ParentID     string  `json:"parent_id,omitempty"`
	SafetyWay    int     `json:"safety_way"`
	RegisterWay  int     `json:"register_way"`
	Secrecy      int     `json:"secrecy"`
	Longitude    float64 `json:"longitude,omitempty"`
	Latitude     float64 `json:"latitude,omitempty"`
	PTZType      int     `json:"ptz_type,omitempty"`
}

// AlarmInfo 告警信息
type AlarmInfo struct {
	DeviceID   string `json:"device_id"`
	AlarmType  string `json:"alarm_type"`
	AlarmLevel string `json:"alarm_level"`
	AlarmTime  string `json:"alarm_time"`
}

// ====== MANSCDP XML 解析器 ======

// ParseCatalogueResponse 解析 MANSCDP 目录响应
func ParseCatalogueResponse(xmlData string) ([]ChannelInfo, error) {
	type MANSCDPItem struct {
		DeviceID     string  `xml:"DeviceID"`
		Name         string  `xml:"Name"`
		Manufacturer string  `xml:"Manufacturer"`
		Model        string  `xml:"Model"`
		Owner        string  `xml:"Owner"`
		CivilCode    string  `xml:"CivilCode"`
		Address      string  `xml:"Address"`
		Parental     int     `xml:"Parental"`
		ParentID     string  `xml:"ParentID"`
		SafetyWay    int     `xml:"SafetyWay"`
		RegisterWay  int     `xml:"RegisterWay"`
		Secrecy      int     `xml:"Secrecy"`
		Status       string  `xml:"Status"`
		Longitude    float64 `xml:"Longitude"`
		Latitude     float64 `xml:"Latitude"`
		PTZType      int     `xml:"PTZType"`
	}
	type MANSCDPResponse struct {
		XMLName    xml.Name      `xml:"Response"`
		CmdType    string        `xml:"CmdType"`
		SN         string        `xml:"SN"`
		DeviceID   string        `xml:"DeviceID"`
		SumNum     int           `xml:"SumNum"`
		DeviceList []MANSCDPItem `xml:"DeviceList>Item"`
	}

	var resp MANSCDPResponse
	if err := xml.Unmarshal([]byte(xmlData), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal MANSCDP XML: %w", err)
	}
	if !strings.EqualFold(resp.CmdType, "Catalog") {
		return nil, fmt.Errorf("unexpected CmdType: %s", resp.CmdType)
	}

	channels := make([]ChannelInfo, 0, len(resp.DeviceList))
	for _, it := range resp.DeviceList {
		channels = append(channels, ChannelInfo{
			DeviceID:     it.DeviceID,
			Name:         it.Name,
			Manufacturer: it.Manufacturer,
			Model:        it.Model,
			Owner:        it.Owner,
			CivilCode:    it.CivilCode,
			Address:      it.Address,
			Parental:     it.Parental,
			ParentID:     it.ParentID,
			SafetyWay:    it.SafetyWay,
			RegisterWay:  it.RegisterWay,
			Secrecy:      it.Secrecy,
			Status:       it.Status,
			Longitude:    it.Longitude,
			Latitude:     it.Latitude,
			PTZType:      it.PTZType,
		})
	}
	return channels, nil
}

// ParseAlarmResponse 解析 MANSCDP 告警通知
func ParseAlarmResponse(xmlData string) (*AlarmInfo, error) {
	type MANSCDPAlarm struct {
		DeviceID   string `xml:"DeviceID"`
		AlarmType  string `xml:"AlarmType"`
		AlarmLevel string `xml:"AlarmLevel"`
		Time       string `xml:"Time"`
	}
	type MANSCDPNotify struct {
		XMLName xml.Name     `xml:"Notify"`
		CmdType string       `xml:"CmdType"`
		SN      string       `xml:"SN"`
		Alarm   MANSCDPAlarm `xml:"Alarm"`
	}
	var notify MANSCDPNotify
	if err := xml.Unmarshal([]byte(xmlData), &notify); err != nil {
		return nil, fmt.Errorf("unmarshal MANSCDP alarm: %w", err)
	}
	if !strings.EqualFold(notify.CmdType, "Alarm") {
		return nil, fmt.Errorf("unexpected CmdType: %s", notify.CmdType)
	}
	return &AlarmInfo{
		DeviceID:   notify.Alarm.DeviceID,
		AlarmType:  notify.Alarm.AlarmType,
		AlarmLevel: notify.Alarm.AlarmLevel,
		AlarmTime:  notify.Alarm.Time,
	}, nil
}

// CatalogueQueryXML 构造 MANSCDP 目录查询指令 XML
func CatalogueQueryXML(sn string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Query>
<CmdType>Catalog</CmdType>
<SN>%s</SN>
<DeviceID>all</DeviceID>
</Query>`, sn)
}

// formatPlaybackTime 格式化回放时间为 GB28181 标准格式
func formatPlaybackTime(t time.Time) string {
	return t.Format("2006-01-02T15:04:05")
}

// MapChannelStatus 映射 GB28181 状态到 Device 状态
func MapChannelStatus(gbStatus string) string {
	switch gbStatus {
	case "ON":
		return model.DeviceStatusOnline
	case "OFF":
		return model.DeviceStatusOffline
	default:
		return model.DeviceStatusUnknown
	}
}
