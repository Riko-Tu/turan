package kube

var Deployment = `
apiVersion: v1
kind: Service
metadata:
  name: tefs
  labels:
    app: tefs-laboratory
spec:
  type: NodePort
  ports:
    - port: 50051
      name: grpc-port
      targetPort: 50051
      nodePort: 32500
  selector:
    app: tefs-laboratory
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tefs-laboratory
spec:
  replicas: 1
  selector:
    matchLabels:
      app: tefs-laboratory
  template:
    metadata:
      labels:
        app: tefs-laboratory
        version: v1
    spec:
      containers:
        - name: tefs-laboratory
          image: ccr.ccs.tencentyun.com/tefs/laboratory:latest
          imagePullPolicy: Always # 每次创建 Pod 都会重新拉取一次镜像
#          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: 50051
          env:
            - name: TEFS_ISDEVELOPMENT
              value: "true"
            - name: TEFS_LOG_MAXCOUNT
              value: "10"
            - name: TEFS_TENCENTCLOUD_INSTANCEID
              valueFrom:
                secretKeyRef:
                  name: tefs-laboratory-config
                  key: InstanceId
            - name: TEFS_TENCENTCLOUD_INSTANCEPASSWORD
              valueFrom:
                secretKeyRef:
                  name: tefs-laboratory-config
                  key: InstancePassWord
            - name: TEFS_TENCENTCLOUD_SECRETID
              valueFrom:
                secretKeyRef:
                  name: tefs-laboratory-config
                  key: SecretId
            - name: TEFS_TENCENTCLOUD_SECRETKEY
              valueFrom:
                secretKeyRef:
                  name: tefs-laboratory-config
                  key: SecretKey
            - name: TEFS_TENCENTCLOUD_REGION
              valueFrom:
                secretKeyRef:
                  name: tefs-laboratory-config
                  key: Region
            - name: TEFS_TENCENTCLOUD_APPID
              valueFrom:
                secretKeyRef:
                  name: tefs-laboratory-config
                  key: AppId
            - name: TEFS_TENCENTCLOUD_COS_BUCKET
              valueFrom:
                secretKeyRef:
                  name: tefs-laboratory-config
                  key: CosBucket
            - name: TEFS_TENCENTCLOUD_SECURITYGROUPID
              valueFrom:
                secretKeyRef:
                  name: tefs-laboratory-config
                  key: SecurityGroupId
            - name: TEFS_TENCENTCLOUD_ZONE
              valueFrom:
                secretKeyRef:
                  name: tefs-laboratory-config
                  key: Zone
            - name: TEFS_TENCENTCLOUD_ACCOUNT
              valueFrom:
                secretKeyRef:
                  name: tefs-laboratory-config
                  key: Account
            - name: TEFS_TENCENTCLOUD_PROJECTID
              valueFrom:
                secretKeyRef:
                  name: tefs-laboratory-config
                  key: ProjectId
      imagePullSecrets:
        - name: qcloudregistrykey
`