#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'
EXPERIMENTAL=true

echo "== Installing dependencies"
apt-get update
apt-get -y install autoconf automake build-essential libass-dev libfreetype6-dev \
  libtheora-dev libtool libvorbis-dev pkg-config texinfo zlib1g-dev \
  yasm wget cmake mercurial libx264-dev

rm -rf $HOME/.ffmpeg_sources && mkdir -p $HOME/.ffmpeg_sources
rm -rf $HOME/.ffmpeg_build && mkdir -p $HOME/.ffmpeg_build

mkdir -p $HOME/.bin/
rm -f $HOME/.bin/{ffmpeg,ffprobe,ffplay,ffserver,vsyasm,x264,x265,yasm,ytasm}

echo "== Installing libx265..."
cd "$HOME/.ffmpeg_sources"
hg clone https://bitbucket.org/multicoreware/x265
cd "$HOME/.ffmpeg_sources/x265/build/linux"
PATH="$HOME/.bin:$PATH" cmake -G "Unix Makefiles" -DCMAKE_INSTALL_PREFIX="$HOME/.ffmpeg_build" -DENABLE_SHARED:bool=off ../../source
make
make install
#make distclean

echo "== Install libfdk-aac..."
cd $HOME/.ffmpeg_sources
wget -O fdk-aac.tar.gz https://github.com/mstorsjo/fdk-aac/tarball/master
tar xzvf fdk-aac.tar.gz
cd mstorsjo-fdk-aac*
autoreconf -fiv
./configure --prefix="$HOME/.ffmpeg_build" --disable-shared
make
make install
#make distclean

echo "== Installing ffmpeg"
cd $HOME/.ffmpeg_sources
if [ EXPERIMENTAL ]; then
    git clone https://git.ffmpeg.org/ffmpeg.git ffmpeg
else
    wget http://ffmpeg.org/releases/ffmpeg-snapshot.tar.bz2
    tar xjvf ffmpeg-snapshot.tar.bz2
fi
cd ffmpeg
PATH="$HOME/.bin:$PATH" PKG_CONFIG_PATH="$HOME/.ffmpeg_build/lib/pkgconfig" ./configure \
  --prefix="$HOME/.ffmpeg_build" \
  --pkg-config-flags="--static" \
  --extra-cflags="-I$HOME/.ffmpeg_build/include" \
  --extra-ldflags="-L$HOME/.ffmpeg_build/lib" \
  --bindir="$HOME/.bin" \
  --extra-libs=-lpthread \
  --enable-gpl \
  --enable-libass \
  --enable-libfdk-aac \
  --enable-libfreetype \
  --enable-libtheora \
  --enable-libvorbis \
  --enable-libx264 \
  --enable-libx265 \
  --enable-nonfree
PATH="$HOME/.bin:$PATH" make
make install
make distclean
hash -r
