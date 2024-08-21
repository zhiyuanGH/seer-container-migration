#This script contains some commands for doing exp
docker run -v /mnt/nfs_share:/data -d --name loooper3 --security-opt seccomp:unconfined ghcr.io/stargz-containers/golang:1.18-esgz /bin/sh -c 'i=0; while true; do echo $i; i=$(expr $i + 1); sleep 1; done'
findmnt /mnt/nfs_share
sudo mount 192.168.116.148:/srv/nfs/share /mnt/nfs_share
sudo umount /mnt/nfs_share
source /etc/profile