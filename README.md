# Change retentions in kafka topics
### This repo need to build image for change retention in kafka.
Becaurse of NT stand with PROD retentions in kafka topics, kafka take a lot of disk memory. This projekt need for build image for cronjob in k8s for change retentions. 

In `change-kafka-retention.go` use [github.com/IBM/sarama](https://github.com/IBM/sarama?tab=readme-ov-file) plugin for connecting to kafka. Take ENV-s from global env, connect to kafka, change retention/ms and delete.retentions.ms in topics if retention or delete.retention is lower then RETENTION_MS and DELETE_RETENTION_MS respectively. If retention ms is lower, but delete.retention is not, then retention will be change but not a delete.retention and vice versa. By default all topics will be find for checking, but you can indicate TOPICS env and script will be look only on tooopics in env.

#####TODO 
add telegramm messaging if need

Script bin file frpm `change-kafka-retention.go` need global envs for start:
- KAFKA_IP - ip of kafka broker, `localhost` by default
- KAFKA_PORT - kafka broker port, `9092` by default
- RETENTION_MS - retention for topics, `1900000` (30m) by default
- DELETE_RETENTION_MS - delete retention for topics, `1900000` (30m) by default
- TOPICS - list of topic for changing retention, by default will chanche retentions in all topics
- TELEGRAM_TOKEN - tg token, by default ""
- TELEGRAM_CHAT_ID - chat id, by default ""

#### Examplle of local run 
```bash
go build -mod=vendor

export KAFKA_IP="127.0.0.2" \
export KAFKA_PORT="9092" \
export RETENTION_MS="2000000" \
export DELETE_RETENTION_MS="2000000" \
export TELEGRAM_TOKEN="123123123123123" \
export TELEGRAM_CHAT_ID="432432423" \

./kafka-change-retantion
```

