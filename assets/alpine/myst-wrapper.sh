#!/bin/sh

function terminate()
{
    logger terminate..
    jobs -p |xargs -r kill -TERM
    exit
}
trap terminate SIGTERM

while true
do
    l=`find /root/.mysterium/keystore -type f | wc -l`
    if [ $l -ge 2 ]
    then
      logger spawn node..
      /root/node/myst $@ &
      pid=$! 

       # exit if there's no child process 
      if [ -z $pid ]
      then
        exit 1
      fi
      break
    fi
    echo Waiting for keystore..
    sleep 5
done

wait $pid
