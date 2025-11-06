#! /bin/sh

if [ $# -ge 1 ]
then
    image=$1
else
    echo "Enter an image name"
    read image
    if [ -z "$image" ]
    then
        echo "Enter an image name!"
        exit
    fi
fi

echo "image: $image"
if [ ! -f $image ]
then
    echo "$image does not exist"
    exit
fi

if [ ! -d icon.iconset ]
then
    mkdir icon.iconset
fi

sips -z 16 16 $image --out icon.iconset/icon_16x16.png
sips -z 32 32 $image --out icon.iconset/icon_32x32.png
sips -z 64 64 $image --out icon.iconset/icon_64x64.png
sips -z 128 128 $image --out icon.iconset/icon_128x128.png
sips -z 256 256 $image --out icon.iconset/icon_256x256.png

iconutil -c icns icon.iconset

# "Now this is not the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942
