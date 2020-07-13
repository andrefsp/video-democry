#! /bin/bash

cd /opt/vid/go

LISTEN_PORT=443 V_HOSTNAME=local.democry.org SSL=true V_PATH=$PWD  RELAY_ADDR=178.62.29.167 nohup ./democry
