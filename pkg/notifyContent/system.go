package notifyContent

// 创建项目成功通知
var CreateProjectSuccess = `
您已成功创建项目【%s】,并默认您为该项目的管理员。
`

// 成员变更通知:添加
var MembersChangeAdd = `
成员【%s】已加入项目【%s】，账号角色为 %s。
`

// 成员变更:禁用
var MembersChangeReduce = `
成员【%s】已被管理员移出项目。
`

// 成员审核
var MembersVerify = `
用户【%s】申请加入项目【%s】，等待您的审核。
`

// 权限变更
var AuthorityChange = `
成员【%s】由%s变更为%s。
`

// 加入项目通知:成功
var AddProjectSuccess = `
您已成功加入项目【%s】，账号角色为%s。
`

// 加入项目通知:失败
var AddProjectFailed = `
（失败）您加入项目【%s】的申请未通过项目管理员审核，您可以与项目管理员联系并再次发送申请。
`

// VASP License审核通知：成功
var LicenseVerifySuccess = `
您申请的VASP License（所属机构：%s）已通过审核，点击查看详细信息。
`

// VASP License审核通知：失败
var LicenseVerifyFailed = `
您申请的VASP License（所属机构：%s）未通过审核，失败原因为：%s ，您可根据上述原因补充申请材料并重新提交申请。
`

// 实验计算
var ExperimentRet = `
您项目【%s】实验【%s】%s。
`