package model

import (
	"TEFS-BE/pkg/database"
	"fmt"
	"github.com/jinzhu/gorm"
)

type UserProject struct {
	Id            int64
	UserId        int64
	ProjectId     int64
	VaspLicenseId int64
	Role          int64
	Status        int64
	CreateAt      int64
	UpdateAt      int64
	DeleteAt      int64
}

// 用户组织项目结构体，接收连表查询结果
type UserOrganizationProject struct {
	Id                  int64
	ProjectId           int64
	ProjectName         string
	Role                int64
	VaspLicenseId       int64
	Organization        string
	Domain              string
	CreateAt            int64
	UpdateAt            int64
	TencentCloudAccount string
}

// 项目成员结构体，接收user_project user 连表查询结果
type ProjectUser struct {
	Id          int64
	UserId      int64
	UserName    string
	Role        int64
	LastLoginAt int64
	CreateAt    int64
}

type MyProject struct {
	ProjectId      int64
	VaspLicenseId  int64
	Role           int64
	CreateUserId   int64
	Name           string
	InvitationCode string
	Status         int64
	CreateAt       int64
}

type UserProjectService struct {
}

func (UserProject) TableName() string {
	return "user_project"
}

// id获取记录
func (up UserProjectService) Get(id int64) (*UserProject, error) {
	userProject := &UserProject{}
	db := database.GetDb()
	err := db.Where("id = ? AND delete_at = 0", id).First(userProject).Error
	if err != nil && err == database.NotFoundErr {
		err = nil
	}
	return userProject, err
}

// 用户id和项目id获取用户项目记录
func (up UserProjectService) GetByUserIdAndProjectId(userId, projectId int64) (*UserProject, error) {
	db := database.GetDb()
	userProject := &UserProject{}
	err := db.Where("user_id = ? AND project_id = ?", userId, projectId).First(userProject).Error
	if err == database.NotFoundErr {
		err = nil
	}
	return userProject, err
}

// 更新
func (up UserProjectService) Update(userProject *UserProject, ups map[string]interface{}) *gorm.DB {
	db := database.GetDb()
	return db.Model(userProject).Update(ups)
}

// 内连接查询vasp_license project user_project, cloud_env 四张表
func (up UserProjectService) GetOrganizationProjectList(userId int64) ([]UserOrganizationProject, error) {
	db := database.GetDb()
	sql := "SELECT tmp.id,tmp.project_id,tmp.role,tmp.create_at,tmp.update_at,vasp_license.id as vasp_license_id," +
		"vasp_license.organization,vasp_license.domain,project.`name` as project_name,cloud_env.cloud_secret_id AS " +
		"tencent_cloud_account FROM (SELECT * FROM user_project WHERE user_id=? AND delete_at=0 AND `status`=1) as " +
		"tmp INNER JOIN project ON tmp.project_id = project.id INNER JOIN vasp_license ON tmp.vasp_license_id = " +
		"vasp_license.id INNER JOIN cloud_env ON project.cloud_env_id=cloud_env.id WHERE (project.`status`=3 AND " +
		"project.delete_at=0 AND vasp_license.`status`=3 AND vasp_license.delete_at=0 AND cloud_env.`status`=3 AND " +
		"cloud_env.delete_at=0)"
	var UserOrganizationProjectList []UserOrganizationProject
	if err := db.Raw(sql, userId).Scan(&UserOrganizationProjectList).Error; err != nil {
		return nil, err
	}
	return UserOrganizationProjectList, nil
}

// 连表查询（user_project, user）
func (up UserProjectService) GetProjectUsers(sortField, sort string, projectId, offset, limit int64) ([]ProjectUser,
	int64, error) {

	db := database.GetDb()
	var total int64
	var projectUserList []ProjectUser

	fields := "up.id,up.user_id,`user`.name as user_name,up.role,`user`.last_login_at,up.create_at "
	count := "count(*)"
	sql := "select %s from (SELECT * from user_project WHERE project_id=? and delete_at=0 and status=1) " +
		"as up INNER JOIN `user` on up.user_id=`user`.id"
	var err error
	if err = db.Raw(fmt.Sprintf(sql, count), projectId).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := fmt.Sprintf("%s %s", sortField, sort)
	if err = db.Raw(fmt.Sprintf(sql, fields),
		projectId).Order(order).Limit(limit).Offset(offset).Scan(&projectUserList).Error; err != nil {
		return nil, 0, err
	}
	return projectUserList, total, nil
}

func (up UserProjectService) GetUserProjectList(userId, offset, limit int64) ([]MyProject, int64, error) {
	db := database.GetDb()
	var total int64
	var myProjects []MyProject

	fields := "up.role, p.vasp_license_id, p.id as project_id,p.user_id as create_user_id,p.name,p.invitation_code,p.status,p.create_at"

	sql := "select %s from (select project.* from user_project " +
		"RIGHT JOIN project ON user_project.project_id=project.id " +
		"where user_project.user_id=? and user_project.delete_at=0 and user_project.`status`=1 UNION " +
		"select * from project where user_id=? and delete_at=0 and status !=1) as p LEFT JOIN " +
		"(SELECT project_id,role from user_project where user_id=? and `status`=1 and delete_at=0) as up " +
		"on p.id=up.project_id"

	countSql := fmt.Sprintf(sql, "count(*)")
	if err := db.Raw(countSql, userId, userId, userId).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	querySql := fmt.Sprintf(sql, fields)
	db.Raw(querySql, userId, userId,
		userId).Offset(offset).Order("create_at DESC").Limit(limit).Scan(&myProjects)
	return myProjects, total, nil
}

func (up UserProjectService) GetLastUserProject(userId int64) Project {
	db := database.GetDb()
	var projects []Project
	sql := "select project.* from (select project_id,create_at from user_project where user_id=? and delete_at=0 " +
		"and status=1) t INNER JOIN project ON t.project_id=project.id ORDER BY t.create_at DESC limit 1"
	db.Raw(sql, userId).Scan(&projects)
	if len(projects) > 0 {
		return projects[0]
	} else {
		return Project{}
	}
}

func (up UserProjectService) GetProjectCloudEnvIp(userId, projectId int64) (*CloudEnv, error) {
	db := database.GetDb()
	cloudEnv := &CloudEnv{}
	sql := "SELECT cloud_env.* FROM " +
		"(SELECT project_id FROM user_project WHERE user_id=? AND project_id=? AND `status`=1 AND delete_at=0) as u " +
		"INNER JOIN " +
		"(SELECT id,cloud_env_id FROM project WHERE id=? AND `status`=3 AND delete_at=0) as p " +
		"ON u.project_id=p.id INNER JOIN cloud_env ON p.cloud_env_id=cloud_env.id"
	if err := db.Raw(sql, userId, projectId, projectId).Scan(cloudEnv).Error; err != nil {
		return nil, err
	}
	return cloudEnv, nil
}
