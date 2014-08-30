#!/bin/bash
echo Entering Script

GOPATH='/root/go'
PROJECTBASEPATH=$GOPATH'/src/github.com/matYang/goPhantom'
CLEANERPATH=$PROJECTBASEPATH'/seoCleaner'
SEOSERVERPATH=$PROJECTBASEPATH'/seoServer'
RESOURCEPATH=$PROJECTBASEPATH'/resource'
DESTINATIONPATH='~/goPhantom'

cd $PROJECTBASEPATH
while true; do
    read -p "Do you wish to perform [git pull origin master]?" yn
    case $yn in
        [Yy]* ) git pull origin master; break;;
        [Nn]* ) break;;
        * ) echo "Please answer yes or no.";;
    esac
done

#Go build literally takes no time, so force build would be apropriate
cd $CLEANERPATH
go install

cd $SEOSERVERPATH
go install

sudo cp $RESOURCEPATH/* $DESTINATIONPATH
cd $DESTINATIONPATH

echo killing the seoCleaner process
sudo ps -ef | grep 'seoCleaner' | awk '{print $2}' | xargs kill

echo killing the seroServer process
sudo ps -ef | grep 'seoServer' | awk '{print $2}' | xargs kill

sudo seoCleaner > seoCleaner.log &
sudo seoServer > seoServer.log &