package notifyContent

var EmailVerifyCode = `
<p>感谢您使用腾讯量子实验室TEFS服务，您的邮件验证码为%s（十分钟有效），请勿泄露给他人。</p>

<div align="right">腾讯量子实验室TEFS项目组</div>
`

var EmailVaspLicenseVerifySuccess = `
<p>恭喜您申请的VASP License已通过审核，License 详细信息如下：</p>

<p>VASP License ID: %d</p>

<p>License所属机构：%s</p>

<p>License申请时间：%s</p>

<p>点击链接进入官网 https://tefscloud.com 继续创建项目。</p>

<div align="right">腾讯量子实验室TEFS项目组</div>
`

var EmailVaspLicenseVerifyFailed = `
<p>您申请的VASP License未通过审核，License 详细信息如下：</p>

<p>VASP License ID: %d</p>

<p>License所属机构：%s</p>

<p>License申请时间：%s</p>

<p>您可以进入官网 https://tefscloud.com 继续完善信息重新申请。</p>

<div align="right">腾讯量子实验室TEFS项目组</div>
`

var EmailAddProjectSuccess = `
<p>恭喜您加入项目[%s]的申请已通过审核，您可以点击链接 https://tefscloud.com 进入项目创建实验啦！</p>

<p>感谢您使用腾讯量子实验室TEFS服务，如有问题，请发送邮件至 %s。</p>

<div align="right">腾讯量子实验室TEFS项目组</div>
`

var EmailExperimentDone = `
<p>您在项目[%s]运行的实验[%s]%s，实验详情如下：</p>

<p>实验名称：%s</p>

<p>开始运行时间：%s</p>

<p>计算时长：%s</p>

<p>您可以进入项目 https://tefscloud.com 查看实验详情。</p>

<p>感谢您使用腾讯量子实验室TEFS服务，如有问题，请发送邮件至 %s。</p>

<p>本邮件提醒可在TEFS服务个人中心页面关闭。</p>


<div align="right">腾讯量子实验室TEFS项目组</div>
`

var EmailCollaborateUrl = `
<p>腾讯量子实验室TEFS用户%s邀请您%s文档<a href="%s">%s</a>，请勿泄露。</p>

<p>若无法打开，请复制以下链接到浏览器地址栏打开：</p>

<p>%s</p>

<div align="right">腾讯量子实验室TEFS项目组</div>
`