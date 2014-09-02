#!/bin/bash
echo Entering Script

export GOPATH='/root/go'
PROJECTBASEPATH=$GOPATH'/src/github.com/matYang/goPhantom'
EXECUTABLEPATH=$GOPATH'/bin'

CLEANERMODULE='seoCleaner'
SEOSERVERMODULE='seoServer'

CLEANERPATH=$PROJECTBASEPATH'/'$CLEANERMODULE
SEOSERVERPATH=$PROJECTBASEPATH'/'$SEOSERVERMODULE
RESOURCEPATH=$PROJECTBASEPATH'/resource'
DESTINATIONPATH=$HOME'/goPhantom'

CLEANEREXECUTABLE=$EXECUTABLEPATH'/'$CLEANERMODULE
SEOSERVEREXECUTABLE=$EXECUTABLEPATH'/'$SEOSERVERMODULE

cd $PROJECTBASEPATH
while true; do
    read -p "Do you wish to perform [git pull origin master]?" yn
    case $yn in
        [Yy]* ) git pull origin master; break;;
        [Nn]* ) break;;
        * ) echo "Please answer yes or no.";;
    esac
done

while true; do
    read -p "Do you with to proceed with deployment? " yn
    case $yn in
        [Yy]* ) break;;
        [Nn]* ) exit;;
        * ) echo "Please answer yes or no.";;
    esac
done

#Go build literally takes no time, so force build would be apropriate
cd $CLEANERPATH
/usr/local/go/bin/go install

cd $SEOSERVERPATH
/usr/local/go/bin/go install

sudo cp -r  $RESOURCEPATH/* $DESTINATIONPATH/
cd $DESTINATIONPATH

echo killing the seoCleaner process
sudo ps -ef | grep $CLEANERMODULE | awk '{print $2}' | xargs kill

echo killing the seroServer process
sudo ps -ef | grep $SEOSERVERMODULE | awk '{print $2}' | xargs kill

sudo $CLEANEREXECUTABLE > $CLEANERMODULE'.log' &
sudo $SEOSERVEREXECUTABLE > $SEOSERVERMODULE'.log' &