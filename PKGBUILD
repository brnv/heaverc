# Maintainer: Alexey Baranov <a.baranov@office.ngs.ru>

pkgname=heaverc-ng
pkgver=c08c5d
pkgrel=1
pkgdesc="heaverd-ng client"
arch=('x86_64')
url="http://git.rn/projects/DEVOPS/repos/heaverc/"
license=('unknown')
backup=('etc/heaverc-ng/config.toml')
md5sums=() #generate with 'makepkg -g'
branch="dev"

pkgver() {
	cd $srcdir
	VERSION=`git show-ref | grep $branch | head -n1 | cut -c 1-6`
	printf $VERSION
}

prepare() {
	rm -rf $srcdir/*
	rm -rf $srcdir/.git
}

build() {
	sudo -ua.baranov git clone ssh://git@git.rn/devops/heaverc.git $srcdir
	cd $srcdir
	git checkout $branch
	go build main.go request.go
}

package() {
	mkdir -p $pkgdir/usr/bin/
	mkdir -p $pkgdir/etc/heaverc-ng/

	cp $srcdir/main $pkgdir/usr/bin/heaverc-ng
	cp $srcdir/config.toml $pkgdir/etc/heaverc-ng/
}
