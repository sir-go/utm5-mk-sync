#!/usr/bin/env bash

username=
password=
service_url="http://userside.ttnet.ru:15672/api/exchanges/%2F/amq.default/publish"

usage_msg () {
    echo "mikrotik command client"
    echo "usage: ./$(basename ${0}) <cmd>"
    echo ""
    echo "  cmd:"
    echo "    help|-h|--help - show this message"
    echo "    slink_add <login> <account_id> <ip_group> <tariff_link_id> <tih|kor>"
    echo "    slink_change <login> <account_id> <ip_group> <tariff_link_id> <tih|kor>"
    echo "    slink_del <login> <account_id> <tariff_link_id> <tih|kor>"
    echo "    internet_on <login> <account_id> <ip_group> <tih|kor>"
    echo "    internet_off <login> <account_id> <ip_group> <tih|kor>"
    echo ""
    echo "    sync_all"
    echo "    rebalance_q"
}

doRequest () {
    body='{"properties":{},"routing_key":"mk","payload":"'${data}'","payload_encoding":"string"}'
    curl -s -d "${body}" -H "Content-Type: application/json" -X POST ${service_url} -u ${username}:${password} > /dev/null
}

case ${1} in

help|-h|--help) usage_msg
;;

slink_add|\
slink_change)
    cmd=${1}
    login=${2}
    aid=${3}
    ips=${4}
    tlid=${5}
    city=${6}
    data='{\"cmd\": \"'${cmd}'\", \"user\": \"'${login}'#'${aid}'#'${tlid}'\", \"ips\": \"'${ips}'\", \"tlid\": '${tlid}', \"city\": \"'${city}'\"}'
    doRequest
;;

slink_del|\
sync)
    cmd=${1}
    login=${2}
    aid=${3}
    tlid=${4}
    city=${5}
    data='{\"cmd\": \"'${cmd}'\", \"user\": \"'${login}'#'${aid}'#'${tlid}'\", \"city\": \"'${city}'\"}'
    doRequest
;;

internet_on|\
internet_off)
    cmd=${1}
    login=${2}
    aid=${3}
    ips=${4}
    city=${5}
    data='{\"cmd\": \"'${cmd}'\", \"user\": \"'${login}'#'${aid}'\", \"ips\": \"'${ips}'\", \"city\": \"'${city}'\"}'
    doRequest
;;

rebalance_q|\
sync_all)
    cmd=${1}
    data='{\"cmd\": \"'${cmd}'\"}'
    doRequest
;;

show_hash)
    doRequest
;;

*) usage_msg
;;

esac
