# Changelog

## v3.2.8

### Bug Fixes

- 修复 Windows 下凭据管理页面保存 AI 配置时报错 `error parsing arguments: received 9 arguments to method 'main.App.UpdateProfileAIConfig', expected 8` 的问题，原因是 Wails 绑定文件未同步更新导致前端传递参数数量与后端不匹配
