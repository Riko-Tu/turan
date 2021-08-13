package ssh

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"time"
)

// 链接
func Connect(user, password, host string, port int) (*sftp.Client, *ssh.Client, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		sshClient    *ssh.Client
		sftpClient   *sftp.Client
		err          error
	)
	auth = make([]ssh.AuthMethod, 0)
	auth = append(auth, ssh.Password(password))
	clientConfig = &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: 30 * time.Second,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	addr = fmt.Sprintf("%s:%d", host, port)
	if sshClient, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, nil, err
	}
	if sftpClient, err = sftp.NewClient(sshClient); err != nil {
		return nil, nil, err
	}
	return sftpClient, sshClient, nil
}

// 上传文件
func PushFile(sftpClient *sftp.Client, localFilePath, remoteDir string) error {
	srcFile, err := os.Open(localFilePath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := sftpClient.Create(path.Join(remoteDir, filepath.Base(localFilePath)))
	if err != nil {
		return err
	}
	defer dstFile.Close()
	buf := make([]byte, 1024)
	for {
		n, _ := srcFile.Read(buf)
		if n == 0 {
			break
		}
		if n < 1024 {
			buf = buf[0:n]
		}
		_, _ = dstFile.Write(buf)
	}
	return nil
}

// 执行指令
func RemoteCmd(sshClient *ssh.Client, cmd string) error {
	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	return session.Run(cmd)
}

// ssh client
type Cli struct {
	ip       string      //IP地址
	username string      //用户名
	password string      //密码
	port     int         //端口号
	client   *ssh.Client //ssh客户端
}

// 创建ssh client对象
func New(ip string, username string, password string, port ...int) *Cli {
	cli := new(Cli)
	cli.ip = ip
	cli.username = username
	cli.password = password
	if len(port) <= 0 {
		cli.port = 22
	} else {
		cli.port = port[0]
	}
	return cli
}

func (c *Cli)GetClient() (*ssh.Client, error) {
	if c.client == nil {
		if err := c.connect(); err != nil {
			return nil, err
		}
	}
	return c.client, nil
}

// ssh client 连接
func (c *Cli) connect() error {
	config := ssh.ClientConfig{
		User: c.username,
		Auth: []ssh.AuthMethod{ssh.Password(c.password)},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 10 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", c.ip, c.port)
	sshClient, err := ssh.Dial("tcp", addr, &config)
	if err != nil {
		return err
	}
	c.client = sshClient
	return nil
}

func (c *Cli) Run(cmd string) (ret []byte, err error) {
	if c.client == nil {
		if err = c.connect(); err != nil {
			return
		}
	}
	session, err := c.client.NewSession()
	if err != nil {
		return
	}
	defer session.Close()
	return session.CombinedOutput(cmd)
}

// ssh client执行带交互的指令
func (c *Cli) RunTerminal(shell string, readWrite io.ReadWriter, session *ssh.Session) error {
	if c.client == nil {
		if err := c.connect(); err != nil {
			return err
		}
	}
	defer session.Close()

	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer terminal.Restore(fd, oldState)

	session.Stdout = readWrite
	session.Stderr = readWrite
	session.Stdin = readWrite

	termWidth, termHeight, err := terminal.GetSize(fd)
	if err != nil {
		return err
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := session.RequestPty("xterm", termHeight, termWidth, modes); err != nil {
		return err
	}
	session.Run(shell)
	return nil
}
