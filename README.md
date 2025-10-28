# go(au)dible

:-)

## Debugging/Infos

* Kernel info
  * `uname -r`
  * config `gzip -d /proc/config.gz -c  | less`
  * kernel parameters: `cat /proc/cmdline`

## TODOs

* change io.Reader to io.ReadSeeker interface in WriteCtx
* web interface
  * move gokrazy web interface to 1080 with autoredirect to 1443 and a self-signed cert
    * "HTTPPORT": "1080"
    * "HTTPSPORT": "1443"
    * "UseTLS": "self-signed"
  * run new interface on 443 with self signed certificate (tls/ssl is needed for websocket (at least in some? browsers))
    * generate cert: https://go.dev/src/crypto/tls/generate_cert.go
    * redirect: https://stackoverflow.com/a/63590299
    * redirect: https://stackoverflow.com/questions/37536006/how-do-i-rewrite-redirect-from-http-to-https-in-go
    * redirect: https://gist.github.com/d-schmidt/587ceec34ce1334a5e60
  * play/pause songs
  * save player state on /perm to survive reboots
  * upload songs
  * delete songs

* add reading commands via unix socket for debugging
* implement Previous() reset of currently played track
  * introduce a (percentage) threshold of the file being played
  * when this threshold is not yet reached, change the track
  * otherwise: reset the current track's offset.
* implement recursive file/dir watch and update Player.audioSourceList
  * e.g via https://github.com/fsnotify/fsnotify/issues/18#issuecomment-3109424560
* gpio: implement long press functions
  * fast forward,
  * fast backword
* webcam qr code module
  * decide: via button push or e.g. one webcam shot per second check?
  * see also https://github.com/makiuchi-d/gozxing
* further circuitry stuff:
  * jack plug (klinkenstecker) via gpio (the raspberry pi zero 2w does not have a dedicated phone connector)
    * https://learn.adafruit.com/adding-basic-audio-ouput-to-raspberry-pi-zero?view=all
    * https://raspberrypi.stackexchange.com/questions/49600/how-to-output-audio-signals-through-gpio
    * https://wiki.batocera.org/audio_via_gpio_rpi_only
  * flip switch for power off/on
  * webcam via usb
* fritzing of the hardware setup
