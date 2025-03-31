package version

import (
	"fmt"
	"runtime"
)

var (
	// Version 当前版本号
	Version = "1.0.0"

	// BuildTime 构建时间
	BuildTime = "unknown"

	// GitCommit Git提交哈希
	GitCommit = "unknown"

	// GoVersion Go版本
	GoVersion = runtime.Version()
)

// Info 版本信息结构体
type Info struct {
	Version   string
	BuildTime string
	GitCommit string
	GoVersion string
}

// GetVersion 获取版本信息
func GetVersion() *Info {
	return &Info{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
		GoVersion: GoVersion,
	}
}

// String 返回版本信息的字符串表示
func (v *Info) String() string {
	return fmt.Sprintf("Version: %s, BuildTime: %s, GitCommit: %s, GoVersion: %s",
		v.Version, v.BuildTime, v.GitCommit, v.GoVersion)
}
