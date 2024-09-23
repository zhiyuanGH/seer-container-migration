#This script contains some commands for doing exp
docker run -v /mnt/nfs_share:/data -d --name loooper3 --security-opt seccomp:unconfined ghcr.io/stargz-containers/golang:1.18-esgz /bin/sh -c 'i=0; while true; do echo $i; i=$(expr $i + 1); sleep 1; done'
findmnt /mnt/nfs_share
sudo mount 192.168.116.148:/srv/nfs/share /mnt/nfs_share
sudo umount /mnt/nfs_share
source /etc/profile
protoc --go_out=. --go-grpc_out=. proto/container.proto
docker run --name cnn -v /mnt/nfs_share:/data 192.168.1.102:5000/mnist-rnn-image:org python3 -u main.py --batch-size 64 --test-batch-size 1000 --epochs 3 --lr 0.1 --gamma 0.7 --log-interval 1 --save-model
