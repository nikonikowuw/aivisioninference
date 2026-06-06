// Package buildinfo 集中存放编译时注入的版本信息。
// 通过 -ldflags 在构建时注入，避免 service 层 import main 形成循环依赖。
package buildinfo

// Version 应用版本号（构建时注入），默认 "dev"
var Version = "dev"

// BuildTime 编译时间（构建时注入），默认 "unknown"
var BuildTime = "unknown"
