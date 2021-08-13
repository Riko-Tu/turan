# TEFS
Tencent Elastic First-principles Simulation Wed Backend

## Install
This project use goland and docker. Go check them out if you don't have them locally installed

```
$ go mod vendor && go build -mod=vendor cmd/xxx/xx.go
```

## Usage
Executable files are generated during installation, Or run it directly.

```
$ sercver
$ go build -mod=vendor cmd/xxx/xx.go
```

## Docker
Remote deployment using docker. Three use environments: development, pre release, production. pre release and production use [蓝盾](http://devops.oa.com/console/pipeline/tefs/list/myPipeline). development env build and push:

```
$ docker build -t ccr.ccs.tencentyun.com/tefs/admin .
$ docker build -f Dockerfile-consumer -t ccr.ccs.tencentyun.com/tefs/consumer .
$ docker build -f Dockerfile-laboratory -t ccr.ccs.tencentyun.com/tefs/laboratory .
$ docker login -u Your account --password Your password ccr.ccs.tencentyun.com
$ docker push  ccr.ccs.tencentyun.com/xxx
$ docker logout ccr.ccs.tencentyun.com
```

## Assembly
I believe you have seen that you need to build three docker images. It may be split later. They are respectively: admin(A grpc Web Service), consumer(Experimental queue service), laboratory(The user's own back end is used to interact with Tencent cloud). They can all be found in the cmd file.

## Tree
Project document description.

```
├── cmd                                                 -- project startup entry file
│   ├── admin                                           -- back end services
│   │   ├── consumer                                    -- experimental queue service
│   │   │   └── main.go                                 -- experimental queue service main
│   │   └── service                                     -- a grpc Web Service
│   │       └── service.go                              -- grpc Web Service main
│   ├── creator                                         -- user grpc web service Creator API server
│   │   └── api                                         -- creator restful API
│   │       └── main.go                                 -- creator restful API main
│   └── laboratory                                      -- user grpc web service
│       └── main.go                                     -- user grpc web service main
├── Dockerfile                                          -- admin Dockerfile
├── Dockerfile-consumer                                 -- consumer Dockerfile
├── Dockerfile-laboratory                               -- laboratory Dcokerfile
├── docs                                                -- creator restful API swagger doc 
│   ├── docs.go                                         -- swagger automatic file generation
│   ├── swagger.json                                    -- swagger automatic file generation
│   └── swagger.yaml                                    -- swagger automatic file generation
├── go.mod                                              -- project mod
├── go.sum                                              -- mod sum
├── nginx                                               -- nginx
│   └── nginx.conf                                      -- nginx config file
└── pkg                                                 -- go project pkg
    ├── admin                                           -- admin server pkg
    │   ├── auth                                        -- login auth
    │   │   ├── auth.go                                 -- login base
    │   │   ├── qq.go                                   -- qq login
    │   │   └── wx.go                                   -- weixin login
    │   ├── compute                                     -- compute dir
    │   │   ├── base.go                                 -- compute base
    │   │   ├── oszicar.go                              -- oszicar file handle
    │   │   └── vasp.go                                 -- vasp compute handle
    │   ├── model                                       -- db model
    │   │   ├── cloudEnv.go                             -- cloud env model
    │   │   ├── experiment.go                           -- experiment model
    │   │   ├── notify.go                               -- notify model
    │   │   ├── projectApply.go                         -- project apply model
    │   │   ├── project.go                              -- project model
    │   │   ├── setting.go                              -- setting model
    │   │   ├── tag.go                                  -- tag model
    │   │   ├── user.go                                 -- user model
    │   │   ├── userProject.go                          -- user project model
    │   │   └── vaspLicense.go                          -- vasp license model
    │   ├── proto                                       -- admin grpc proto
    │   │   ├── admin.pb.go                             -- admin grpc automatic file generation
    │   │   └── admin.proto                             -- admin grpc proto file
    │   ├── service                                     -- admin service grpc api handle
    │   │   ├── cloudEnv.go                             -- cloud env GPRC handle
    │   │   ├── codeMsg.go                              -- api definition code msg
    │   │   ├── deploy.go                               -- deploy GPRC handle
    │   │   ├── email.go                                -- email service
    │   │   ├── laboratory.go                           -- laboratory GPRC handle
    │   │   ├── notify.go                               -- notify GPRC handle
    │   │   ├── projectApply.go                         -- project apply GRPC handle
    │   │   ├── project.go                              -- project GRPC handle
    │   │   ├── service.go                              -- base func
    │   │   ├── setting.go                              -- database setting handle
    │   │   ├── tag.go                                  -- experiment tag GRPC handle
    │   │   ├── user.go                                 -- user GRPC handle
    │   │   ├── userProject.go                          -- user project GRPC handle
    │   │   └── vaspLicense.go                          -- vasp license GRPC handle
    │   ├── task                                        -- consumer server handle
    │   │   ├── task.go                                 -- experiment task
    │   │   └── worker.go                               -- experoment worker
    │   └── ws                                          -- websocket API
    │       ├── vaspkit.go                              -- vaspkit websocket API
    │       └── ws.go                                   -- wensocket base func
    ├── cache                                           -- cache
    │   └── redis.go                                    -- redis init
    ├── creator                                         -- user GRPC creator api server 
    │   └── api                                         -- api dir
    │       ├── controllers                             -- controllers
    │       │   ├── account.go                          -- tencent cloud account api
    │       │   ├── base.go                             -- base func
    │       │   ├── cos.go                              -- tencent cloud cos api
    │       │   ├── project.go                          -- tencent cloud project api
    │       │   ├── region.go                           -- tencent cloud region api
    │       │   ├── securityGroup.go                    -- tencent cloud security group api
    │       │   ├── tke.go                              -- tencent cloud tke api
    │       │   └── vpc.go                              -- tencnet cloud vpc api
    │       ├── kube                                    -- k8s kube file
    │       │   ├── deployment.go                       -- deployment kube file
    │       │   └── kube.go                             -- base func
    │       └── router                                  -- api router dir
    │           └── router.go                           -- api router
    ├── database                                        -- mysql database
    │   └── init.go                                     -- database init
    ├── laboratory                                      -- user laboratory GRPC interact with tencent cloud
    │   ├── client                                      -- user laboratory GRPC client
    │   │   └── client.go                               -- user laboratory GRPC client
    │   ├── proto                                       -- grpc porto file
    │   │   ├── laboratory.pb.go                        -- grpc proto go file
    │   │   └── laboratory.proto                        -- proto
    │   └── service                                     -- server
    │       ├── cos.go                                  -- cos GRPC
    │       ├── cvm.go                                  -- cvm GRPC
    │       ├── experiment.go                           -- experiment GRPC
    │       └── service.go                              -- base grpc
    ├── log                                             -- project logger
    │   └── logger.go                                   -- logger
    ├── notifyContent                                   -- notify html
    │   ├── email.go                                    -- email html
    │   ├── sms.go                                      -- sms content
    │   └── system.go                                   -- system content
    ├── tencentCloud                                    -- tencent cloud interaction
    │   ├── account.go                                  -- tencent cloud account
    │   ├── batchCompute                                -- batch compute
    │   │   ├── batchCompute.go                         -- batch compute func
    │   │   └── env.go                                  -- batch env 
    │   ├── cos.go                                      -- cos
    │   ├── credential.go                               -- tencent cloud credential
    │   ├── cvm.go                                      -- cvm
    │   ├── ocr.go                                      -- ocr picture recognition
    │   ├── ses                                         -- ses mail serve
    │   │   ├── models.go                               -- ses model
    │   │   ├── sendEmail.go                            -- send email func
    │   │   └── ses.go                                  -- ses func
    │   ├── sms                                         -- sms
    │   │   └── sms.go                                  -- send sms
    │   ├── tke.go                                      -- tke
    │   └── vpc.go                                      -- vpc
    └── utils                                           -- project encapsulation tools
        ├── common.go                                   -- common tools
        ├── common_test.go                              -- common tools test func
        ├── email                                       -- email
        │   ├── init.go                                 -- email init
        │   └── sendEmail.go                            -- send email func
        └── ssh                                         -- ssh remote server
            └── ssh.go                                  -- ssh func
```