# enforce-ns-annotations

强制为新创建的命名空间添加注解

## 使用方式

本组件使用 `admission-bootstrapper` 安装，首先参照此文档 https://github.com/k8s-autoops/admission-bootstrapper ，完成 `admission-bootstrapper` 的初始化步骤。

然后，部署以下 YAML 即可

```yaml
# create job
apiVersion: batch/v1
kind: Job
metadata:
  name: install-enforce-ns-annotations
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
              value: enforce-ns-annotations
            - name: ADMISSION_IMAGE
              value: autoops/enforce-ns-annotations
            # !!!修改这里!!!
            - name: ADMISSION_ENVS
              value: "CFG_ANNOTATIONS= autoops.enforce-ns-annotations: test"
            - name: ADMISSION_MUTATING
              value: "true"
            - name: ADMISSION_IGNORE_FAILURE
              value: "false"
            - name: ADMISSION_SIDE_EFFECT
              value: "None"
            - name: ADMISSION_RULES
              value: '[{"operations":["CREATE"],"apiGroups":[""], "apiVersions":["*"], "resources":["namespaces"]}]'
      restartPolicy: OnFailure
```

## Credits

Guo Y.K., MIT License
