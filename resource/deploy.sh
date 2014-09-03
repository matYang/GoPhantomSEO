#!/bin/bash
echo Entering Script

#一直找不到方法在bash中调取到GOPATH，故特地写在这里
#每个系统都可能不一样，因人而异，注意不要commit这一行的修改
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

#使用Go的绝对路径，简化Go路径找不到的问题
#根据Go的安装指南，GO的路径在任何unix基础系统上应该在这里
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