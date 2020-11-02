# enforce-qcloud-fixed-ip

自动强制为 StatefulSet 类型开启腾讯云 TKE 固定 Pod IP 功能

腾讯云 TKE 提供 StatefulSet 固定 Pod IP 功能，这个功能非常便捷，尤其是当你想快速迁移传统部署的业务到 TKE 集群的时候。

但是要想启用这个功能，必须在创建 StatefulSet 时，在腾讯云管理台上手动开启这个功能，或者按照文档，在 YAML 中添加指定的字段。

本项目使用了 Kubernetes Admission Webhook 功能，可以自动在创建 StatefulSet 时（无论是通过腾讯云管理台，Rancher 管理台 还是 手动 kubectl 部署 YAML 时）强制开启此功能。

## 前置条件

需要使用 `VPC-ENI` 模式创建腾讯云集群，并且在创建时开启 `固定 Pod IP` 功能

## 使用方式

本组件使用 `admission-bootstrapper` 安装，首先参照此文档 https://github.com/k8s-autoops/admission-bootstrapper ，完成 `admission-bootstrapper` 的初始化步骤。

然后，部署以下 YAML 即可

```yaml
# create job
apiVersion: batch/v1
kind: Job
metadata:
  name: install-enforce-qcloud-fixed-ip
  namespace: autoops
spec:
  template:
    spec:
      serviceAccount: admission-bootstrapper
      containers:
        - name: admission-bootstrapper
          image: autoops/admission-bootstrapper
          env:
            - name: ADMISSION_NAME
              value: enforce-qcloud-fixed-ip
            - name: ADMISSION_IMAGE
              value: autoops/enforce-qcloud-fixed-ip
            - name: ADMISSION_ENVS
              value: ""
            - name: ADMISSION_MUTATING
              value: "true"
            - name: ADMISSION_IGNORE_FAILURE
              value: "false"
            - name: ADMISSION_SIDE_EFFECT
              value: "None"
            - name: ADMISSION_RULES
              value: '[{"operations":["CREATE"],"apiGroups":["apps"], "apiVersions":["*"], "resources":["statefulsets"]}]'
      restartPolicy: OnFailure
```

## Credits

Guo Y.K., MIT License
