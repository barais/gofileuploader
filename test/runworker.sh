#!/bin/bash
folder=`pwd`
#echo "$folder/test.sh $1 $2 $3 $4"
cd $(mktemp -d /tmp/si2XXXX) && $folder/test.sh $1 $2 $3 $4
