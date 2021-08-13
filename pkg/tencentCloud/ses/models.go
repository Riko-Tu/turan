package ses

type Ses struct {
	User     string
	Password string
	Email    string
}

// 发送邮件参数
type SendParams struct {

	// 发件人邮箱 例： 1.无别名：tefs@tefscloud.com 2.添加别名：量子实验室 <tefs@tefscloud.com>
	From string `json:"from"`

	// 收件人邮箱
	To string `json:"to"`

	// 邮件主题
	Subject string `json:"subject"`

	// 邮件文本内容
	Text string `json:"text,omitempty"`

	// 邮件HTML，text和html字段不可以同时使用，如果同时出现，优先使用 html
	Html string `json:"html,omitempty"`

	// 可选，用户回复地址，如果想接收用户回复，可以在这里填入自己可以接 收到邮件的邮箱
	ReplyTo string `json:"replyTo,omitempty"`

	// 邮件附件路径，可以传多个
	Attachment []string `json:"attachment,omitempty"`
}

// 发送邮件响应
type SendResponse struct {
	// API响应code，0代表成功
	Code int `json:"code"`

	// 如果失败，该字段返回错误信息
	Message string `json:"message"`

	//  腾讯云返回标识发送请求的唯一ID，后续的事件将使用bulkId作为KEY来作 为回调
	BulkId string `json:"bulkId,omitempty"`

	// 邮件发送结果信息
	Messages []struct {
		// 收件人地址
		To        string `json:"to"`
		MessageId string `json:"messageId"`
		Status    struct {
			// 发送状态
			Status int `json:"status"`
			//发送状态描述
			Description string `json:"description"`
		} `json:"status"`
	} `json:"messages,omitempty"`
}

// 模板发送参数
type TemplateSendParams struct {

	//  发件人邮箱 示例： YourSenderName
	From string `json:"from"`

	// 收件人邮箱
	To string `json:"to"`

	//  邮件主题
	Subject string `json:"subject"`

	// 事先创建好的模板ID
	TemplateId int `json:"templateId"`

	// 转义后的Json字符串.
	//注意: 在json串中应该保证字段与创建的模板的变量名一一对应，否则会 出现变量无法被替换的情况，
	//比如模板中有两个变量{{first_point_name}}, {{second_point_name}}，
	//那么这里应该传”contentJson”: {\"first_point_name\":\"value1\",\"second_point_name\":\"value2\"}
	ContentJson string `json:"contentJson"`
}

// 查询邮件参数
type QueryParams struct {
	BulkId string `json:"bulkId"`
}

// 查询邮件响应
type QueryResponse struct {
	// 0表示请求已接受
	Code     int    `json:"code"`
	Messages string `json:"messages"`
	Results  []struct {
		BulkId string `json:"bulkId"`
		AppId  string `json:"appId"`
		From   string `json:"from"`
		To     string `json:"to"`
		// 请求时间
		SentTime int64 `json:"sentTime"`
		// 0 表示邮件发送已提交，其它表示错误码
		SentStatus int `json:"sentStatus"`
		// 0 表示初始化状态; 1 表示邮件已附送; 其它 表示错误状态码
		DeliverStatus bool `json:"deliverStatus"`
		// 邮件是否被用户打开
		Opened bool `json:"opened"`
		// 邮件中的链接是否被用户点击
		Clicked bool `json:"clicked"`
		// 递送状态描述
		DeliverDesc string `json:"deliverDesc"`
	} `json:"results"`
}
