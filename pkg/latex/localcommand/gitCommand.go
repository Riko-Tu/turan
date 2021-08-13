package localcommand

import (
	"TEFS-BE/pkg/latex/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	// latex git 版本控制目录
	latexGitDir = "git"
	// latex 读取目录
	latexRawDir = "raw"

	// latex git 操作文件锁
	// 排他，阻塞的文件锁
	latexGitFileLock = ".lock"
)

// 本地git服务
// 通过此结构体调用本地git API
type LatexGit struct {
	*LocalCommand
	Path     string
	FileLock *os.File
}

// 返回一个LatexGit结构体指针
// 用于调用本地封装git API
func NewGitCli(path string) *LatexGit {
	return &LatexGit{
		LocalCommand: &LocalCommand{
			Command: "git",
		},
		Path: filepath.Join(path, latexGitDir),
	}
}

// 上锁
func (g *LatexGit) Lock() error {
	lockPath := filepath.Join(filepath.Dir(g.Path), latexGitFileLock)
	var f *os.File
	var err error
	if !utils.PathExists(lockPath) {
		f, err = os.Create(lockPath)
	} else {
		f, err = os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	}
	if err != nil {
		return err
	}
	g.FileLock = f
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	return nil
}

// 解锁
func (g *LatexGit) Unlock() error {
	defer g.FileLock.Close()
	return syscall.Flock(int(g.FileLock.Fd()), syscall.LOCK_UN)
}

// git 是否已初始化
func (g LatexGit) GitIsInit() (isInit bool) {
	// 是否为目录
	if !utils.IsDir(g.Path) {
		return
	}
	g.Args = []string{
		"rev-parse --is-inside-work-tree",
	}
	ret, _, err := g.OutStdErrRun()
	if err != nil {
		return
	}
	if strings.HasPrefix(ret, "true") {
		isInit = true
		return
	}
	return
}

// latex git 初始化
func (g LatexGit) Init() error {
	if err := g.Lock(); err != nil {
		return err
	}
	defer g.Unlock()
	// 目录是否存在
	// 目录不存在则创建目录
	// 存在但不是目录，则删除，并创建
	// 在创建目录后会copy latex raw 下所有文件 到 latex/git目录下
	rawDir := filepath.Join(filepath.Dir(g.Path), latexRawDir)
	if !utils.PathExists(g.Path) {
		if err := os.MkdirAll(g.Path, 0755); err != nil {
			return err
		}
		if err := CopyDir(rawDir, g.Path); err != nil {
			return err
		}
	} else {
		if !utils.IsDir(g.Path) {
			if err := os.Remove(g.Path); err != nil {
				return err
			}
			if err := os.MkdirAll(g.Path, 0755); err != nil {
				return err
			}
			if err := CopyDir(rawDir, g.Path); err != nil {
				return err
			}
		}
	}

	g.Args = []string{
		"-C",
		g.Path,
		"init",
	}
	_, _, err := g.OutStdErrRun()
	return err
}

// 添加所有git变更
func (g LatexGit) AddAll() error {
	g.Args = []string{
		"-C",
		g.Path,
		"add",
		".",
	}
	_, _, err := g.OutStdErrRun()
	return err
}

// 给定备注，提交一个git版本
func (g LatexGit) Commit(memo string) error {
	g.Args = []string{
		"-C",
		g.Path,
		`commit`,
		`-m`,
		fmt.Sprintf(`"%s"`, memo),
	}
	_, _, err := g.OutStdErrRun()
	return err
}

// 获取最后一次git提交的hash
func (g LatexGit) GetLastHash() (gitHash string, err error) {
	g.Args = []string{
		"-C",
		g.Path,
		`log`,
		`-1`,
		`--pretty=format:"%H"`,
	}
	var result string
	result, _, err = g.OutStdErrRun()
	if err != nil {
		return
	}
	return result, nil
}

// 连续的git指令
// 添加所有变更到暂存区
// 提交
// 获取hash
func (g LatexGit) CommitGetHash(memo string) (gitHash string, err error) {
	if err = g.AddAll(); err != nil {
		return
	}
	if err = g.Commit(memo); err != nil {
		return
	}
	return g.GetLastHash()
}

// 指定git hash版本获取指定版本的目录树
func (g LatexGit) GetDirTreeForHash(commitHash string) (tree []string, err error) {
	g.Args = []string{
		"-C",
		g.Path,
		`ls-tree`,
		`-r`,
		commitHash,
		`--name-only`,
	}
	var ret string
	ret, _, err = g.OutStdErrRun()
	if err != nil {
		return
	}
	blobs := strings.Split(ret, "\n")
	for _, blob := range blobs {
		if len(blob) > 0 {
			tree = append(tree, blob)
		}

	}
	return
}

// 指定git hash版本和文件，获取文件内容
func (g LatexGit) GetFileContentForHash(commitHash, file string) (content string, err error) {
	g.Args = []string{
		"-C",
		g.Path,
		`show`,
		fmt.Sprintf("%s:%s", commitHash, file),
	}
	content, _, err = g.OutStdErrRun()
	return
}

// 提供commit hash 切换到指定版本
func (g LatexGit) Checkout(commitHash string) (err error) {
	g.Args = []string{
		"-C",
		g.Path,
		`checkout`,
		commitHash,
	}
	_, _, err = g.OutStdErrRun()
	return err
}

// latex git 目录同步到raw目录排除，忽略文件列表
var RsyncExcludes = []string{".git", ".gitkeep"}

func LatexRsync(path string) (err error) {
	// latex git 版本控制目录
	gitPath := filepath.Join(path, latexGitDir) + "/"
	// latex 显示目录
	rawPath := filepath.Join(path, latexRawDir) + "/"
	cmd := LocalCommand{
		Command: "rsync",
		Args: []string{
			"-av",
			"--delete",
			gitPath,
			rawPath,
		},
	}
	for _, exclude := range RsyncExcludes {
		cmd.Args = append(cmd.Args, "--exclude", fmt.Sprintf(`"%s"`, exclude))
	}
	_, _, err = cmd.OutStdErrRun()
	return err
}
