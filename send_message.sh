#!/bin/bash

uuid=`uuidgen`
currDate=`date -I`
payload="{
 \"messageType\": [
   \"urn:message:Pipefire:Pipeline:DDRun:TransferFiles\"
 ],
 \"message\": {
   \"task\": \"begin\",
   \"start_date\": \"${currDate}\",
   \"correlationId\": \"${uuid}\"
 }
}"

rabbitmqadmin publish -V / exchange=ddrun routing_key="" payload="$payload";
