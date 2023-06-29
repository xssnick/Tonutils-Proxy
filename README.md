# Tonutils Proxy
[![Based on TON][ton-svg]][ton]

**Your gateway to the new internet**

<img width="294" alt="Screen Shot 2022-11-21 at 18 15 16" src="https://user-images.githubusercontent.com/9332353/203090531-6b37d922-236b-4ff2-857b-dd4965cfa153.png">

This is a user-friendly TON Proxy implementation. It works on any platform with UDP support. It can be used with any internet connection, and any type of ip.  

At this moment client multi-threaded proxy is implemented, reverse-proxy for web3 sites hosting coming soon!

[Join our Telegram group](https://t.me/tonrh) to stay updated! More cool products on this basis are planned.

##### Support project ❤️
If you love this product and want to support its development you can donate any amount of coins to this ton address ☺️
`EQBx6tZZWa2Tbv6BvgcvegoOQxkRrVaBVwBOoW85nbP37_Go`

### Download precompiled version
* [Download Mac M1 (GUI)](https://github.com/xssnick/Tonutils-Proxy/releases/download/v1.3.0/GUI.Mac.M1.Tonutils.Proxy.dmg)
* [Download Mac Intel (GUI)](https://github.com/xssnick/Tonutils-Proxy/releases/download/v1.3.0/GUI.Mac.Intel.Tonutils.Proxy.dmg)
* [Download Windows (GUI)](https://github.com/xssnick/Tonutils-Proxy/releases/download/v1.3.0/GUI-Windows-x64-Tonutils-Proxy.exe)
* [Download Linux (CLI)](https://github.com/xssnick/Tonutils-Proxy/releases/download/v1.3.0/tonutils-proxy-cli-linux-amd64)

You can also find executable for most popular platforms in [Releases](https://github.com/xssnick/Tonutils-Proxy/releases).

If executable is missing for your platform, you can [join our group](https://t.me/tonrh) and ask for it, we may add it to releases list.

### Integrate into your mobile app
You could compile for IOS and Android by yourself using `make build-ios-lib` and `make build-android-lib`. 
To compile for IOS, XCode tools and Mac are required, for Android you need NDK toolchain.

Or you could use precompiled libs.

##### Precompilled
* [Download IOS Library](https://github.com/xssnick/Tonutils-Proxy/releases/download/v1.3.0/ios-lib.zip)
* [Download Android Library](https://github.com/xssnick/Tonutils-Proxy/releases/download/v1.3.0/android-lib.zip)

# How to use

## GUI
#### 1. Open it
You should see:

<img width="303" alt="Screen Shot 2022-11-21 at 18 13 11" src="https://user-images.githubusercontent.com/9332353/203090096-1c03907b-7d29-4be2-83df-d689d2151f08.png">

#### 2. Start it
Click "Start Gateway", it will configure your system automatically and open foundation.ton.

##### If TON sites not opens
If for some reason your system was not autoconfigured or you don't want to reconfigure it, you can enter HTTP proxy address manually in your browser. Follow CLI instructions starting from [section 2](#2-connect-your-browser-to-it). 

HTTP proxy uses `127.0.0.1:8080` address.

## CLI
##### 1. Open it
Double click on it on windows, or run it using terminal on linux/mac.

<img width="572" alt="Screen Shot 2022-11-18 at 17 01 01" src="https://user-images.githubusercontent.com/9332353/202722168-3a41b771-7f61-4ddd-8310-21ae1b2e5b27.png">

HTTP proxy will start on `127.0.0.1:8080` address.

If you are using GUI version, it should configure your system automatically. 
If you are using CLI, or you want to do a manual connection, follow steps below.

##### 2. Connect your browser to it
Open your browser network settings and configure http proxy.
<img width="735" alt="image" src="https://user-images.githubusercontent.com/9332353/202722921-a2f7a92b-c5d8-496d-aaf2-446f01fad0ae.png">

##### 3. Try to connect to some .ton sites
Your proxy is configured now, you can access TON sites!

Lets try to connect to some ton site, for example http://foundation.ton/

<img width="654" alt="Screen Shot 2022-11-18 at 17 41 19" src="https://user-images.githubusercontent.com/9332353/202730383-85bda07b-7bea-4d9c-9aa6-633f76d1cee3.png">

**By the way, this proxy works fine also for Web2 sites, you can seamlessly use it to access both Web2 and Web3.**

<!-- Badges -->
[ton-svg]: https://img.shields.io/badge/Based%20on-TON-blue
[ton]: https://ton.org

### How to build from sources
CLI version has no external dependencies, just [tonutils](https://github.com/xssnick/tonutils-go) and pure Go 🤘
 ```
go build -o ton-proxy cmd/proxy-cli/main.go
 ```
Done!

To build GUI version you need [Wails](https://wails.io/).
