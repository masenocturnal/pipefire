#!/bin/bash

vhost=""
host="localhost"
exchange="BankFileTransfer.Incoming"
uuid=$(uuidgen)
currDate=`date -I`

payload="{ 
  \"messageType\": [   
      \"urn:message:Certegy.DirectDebit.Messaging.Contracts.Payload:BankTransferPayload\" 
  ], 
  \"message\": {   
    \"task\": \"transfer\",   
    \"start_date\": \"$curDate\",   
    \"correlationId\": \"$uuid\" 
  }
}"



payloadNew=$(jq --arg foo $(date -I) --arg bar $(uuidgen) '.message.start_date = $foo | .message.correlationId = $bar'  msg.json )

echo $payloadNew | jq .

rabbitmqadmin publish --host "$host" -V "$vhost" exchange="$exchange" routing_key="" payload="$payloadNew";
