## 本地單獨啟動 golang-worker

`預設本地已有 RabbitMQ、MYSQL 的狀況`
1. cp config.example.yml config.yml
2. vim config.yml
3. 調整 database、rabbitma、log 的相關參數為本機
4. go run main.go

## 完整啟動所有服務(docker-compose)

`透過 docker-compose 啟動 golang-worker、rabbitMQ`
1. cp docker-compose.example.yml docker-compose.yml
2. 調整 service >> golang 區塊的 environment 的各項參數
3. 預設 ELK、LOGSTASH 的 ENABLE 為 0，依需求將之調整為 1，並注意提供 LOG 相關參數
4. docker-compose up -d
5. 如果有異動 golang 原始程式碼，則需要重新打包容器， docker-compose up -d --build

## 部署後的設定

1. 打開 docker-compose.example.yml
2. 打開 gitlab repo 中的 setting >> CI/CD >> Variables
3. 將 docker-compose.example.yml 中的 service >> golang 區塊的 environment 的各項參數維護到其中( KEY 值前需加 K8S_SECRET_)
4. start pipeline