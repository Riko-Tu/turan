package email

import (
	"fmt"
	"gopkg.in/gomail.v2"
	"mime"
	"strings"
)

const alias = "TEFS"

func SendEmail(toEmails []string, subject string, body string, attachment *string) error {
	// 设置邮箱主体
	m := gomail.NewMessage(
		//发送文本时设置编码，防止乱码。
		gomail.SetEncoding(gomail.Base64),
	)
	m.SetHeader("From", m.FormatAddress(emailConfig.user, alias)) // 添加别名
	m.SetHeader("To", toEmails...)                                // 发送给用户(可以多个)
	m.SetHeader("Subject", subject)                               // 设置邮件主题
	m.SetBody("text/html", body)                            // 设置邮件正文

	// 附件文件
	if attachment != nil {
		item := strings.Split(*attachment, "/")
		filename := item[len(item)-1]
		m.Attach(*attachment,
			gomail.Rename(filename), //重命名
			gomail.SetHeader(map[string][]string{
				"Content-Disposition": []string{
					fmt.Sprintf(`attachment; filename="%s"`, mime.QEncoding.Encode("UTF-8", filename)),
				},
			}),
		)
	}

	// 创建SMTP客户端，自动开启SSL，这个时候需要指定TLSConfig
	d := gomail.NewDialer(emailConfig.host, emailConfig.port, emailConfig.user, emailConfig.password)
	//d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	err := d.DialAndSend(m)
	return err
}