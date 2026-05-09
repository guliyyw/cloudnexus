class CWSTcp {
    sendMsg(szMsg) {
        return this.ws.send(szMsg);
    }

    createWebSocket() {
        try {
            this.ws = new WebSocket(this.wsUrl);
            this.ws.parentObj = this;

            this.ws.onopen = function () {
                this.parentObj.heartCheck();
                if (this.parentObj.onConnected)
                    this.parentObj.onConnected();
            }
            this.ws.onmessage = function (e) {
                //this.ws.recieved = 1;
                this.parentObj.heartCheck();
                this.parentObj.onMessage(e.data);
            }
            this.ws.onerror = function (event) {
                console.log("searchcam ws.onerror", event);
                this.parentObj.onError(-1, "连接错误");
                console.log("searchcam 连接错误");
                this.parentObj.reconnect();
            }
            this.ws.onclose = function () {
                console.log("searchcam 连接断开");
                this.parentObj.onError(-1, "连接错误");
                this.parentObj.reconnect();
            }
        } catch (e) {
            console.log('searchcam catch');
            this.reconnect();
        }
    }

    create(url, nTimeout, onConnected, onMessage, onErr) {
        this.wsUrl = url;
        this.lockReconnect = false;//避免重复连接
        this.reconnTimeoutObj = null;
        //this.ws.recieved = 0;
        this.hearttimeout = 30000;
        this.hearttimeoutObj = null;
        this.serverTimeoutObj = null;

        this.onConnected = onConnected;
        this.onMessage = onMessage;
        this.onError = onErr;
        this.createWebSocket();
    }

    reconnect() {
        if (this.lockReconnect) {
            return;
        }
        this.lockReconnect = true;
        //没连接上会一直重连，设置延迟避免请求过多
        this.reconnTimeoutObj && clearTimeout(this.reconnTimeoutObj);
        let self = this;
        this.reconnTimeoutObj = setTimeout(function () {
            self.createWebSocket();
            self.lockReconnect = false;
            console.log('searchcam createSocket');
        }, 1000);
    }

    heartCheck() {
        let self = this;
        this.hearttimeoutObj && clearTimeout(this.hearttimeoutObj);
        this.serverTimeoutObj && clearTimeout(this.serverTimeoutObj);
        this.hearttimeoutObj = setTimeout(function () {
            //这里发送一个心跳，后端收到后，返回一个心跳消息，
            // console.log('searchcam HET');
            self.ws.send("HET");
            self.serverTimeoutObj = setTimeout(function () {
                console.log('searchcam timeout');
                self.ws.close();
            }, self.hearttimeout);

        }, this.hearttimeout)
    }
}


const ExeHostIP = 'ws://127.0.0.1:880';  //保持长连接
class Instance extends CWSTcp {
    constructor() {
        super()
        this.mReady = false
        this.wsTcp = null
		this.onSocketStatusCallback=null
		this.onCommandCallback=null
    }
	Login(userInfo){
		const szMsg = 'LGN'+JSON.stringify(userInfo);
		this.sendMsg(szMsg);
		//返回值：
		//IDS_PARAMERR 参数错误
		//IDS_LOGINFAIL_NOCONNECT';  无法连接服务器
		//IDS_SERVER_ERR_PARAM';  参数错误
		//IDS_SERVER_ERR_NOID';  帐号不存在
		//IDS_ERRTYPE_PWD';  密码错误
	}
    connectDev(dev) {//连接设备  'ADD:
		const szMsg = 'ADD' + JSON.stringify(dev);
		this.sendMsg(szMsg);
		// console.log('ADD:' + szMsg);
		//返回值； 
		//	ADDIDS_ConnectDev: 添加成功
		//	ADDIDS_AlreadyExits:  ADDdevice does already exits: 设备已经添加过，可以直接使用
		//	ADEIDS_PARAMERR:  参数错误
		//	ADEIDS_NODEV： 设备列表里不存在该VUID设备
		
		//设备连接过程中，会返回连接状态
		//通过STA返回的状态值
		//连接过程中：
		//	["IDS_DUAL_AUTHING","正在鉴权"],
		//	["IDS_ConnectDev","连接中..."],
		//  ["IDS_CONNECT_SUCC","连接成功"], 
		
		//连接失败情况：
		//	["IDS_ERR_EXTUSER","当前访问数超过最大数"],
		//	["IDS_ERR_ILLEGALDEV","非法设备"],
		//	["IDS_OFFLINE","不在线"],
		//	["IDS_ERR_ACCESSAUTH","摄像机密码错误"],
		//	["IDS_ERR_FORBIDDEN","密码尝试次数超过限制，等待1小时"],
		//	["IDS_AUTH_TOKEN_ERR","鉴权失败"],
		
		//连接成功情况：
		//	["IDS_AUTH_SUCC","鉴权成功"],
		//  在出视频的情况下，还是返回帧率的信息
		
    }

    disconnectDev(session,devID) {
        const szMsg = 'RMV' + session + ':' +devID;
        this.sendMsg(szMsg);
		//返回值
		//RMV：删除成功
		//RMEdev does not exists：设备不存在
    }
	clearDev(){
		const szMsg = 'CLR';
		this.sendMsg(szMsg);
		//返回值
		//CLR：清空成功
	}
    //激活视频窗口，如果是实时画面的时候bSDPlay为0，SD卡回放界面的时候bSDPlay为1
    ActiveVideo(session,szDev,bSDPlay) {
		const szMsg = 'ACT' + session + ':' + szDev + ':' + bSDPlay;
		this.sendMsg(szMsg);
		//返回值
		//ACT：调用成功		
    }

    Listen(session,devID,bStart) {
        if (bStart) {
            const szMsg = 'LST' + session + ':' + devID;
            this.sendMsg(szMsg);//开始监听
        } else {
            const szMsg = 'LNO' + session + ':' + devID;
            this.sendMsg(szMsg);//停止监听
        }
		//返回值
		//LST：调用成功	drddd
    }

    Talk(session,devID,bStart) {
        if (bStart) this.sendMsg('TST' + session + ':' +  devID);//开始对讲
        else this.sendMsg('TNO' + session + ':' + devID);//停止对讲
		//返回值
		//TST：调用成功	
    }
	//获取SD卡文件列表，nPageIndex:获取第几页， nPageCount:每页获取的文件数
	GetSDFileList(session,devID,nPageIndex,nPageCount){
		const szMsg = 'SDF' + session + ':' + devID + ':' + nPageIndex + ':' + nPageCount;
		this.sendMsg(szMsg);
		//返回值：
		//SDS:指令发送成功； SFN：参数错误或者设备异常
		//在onCommand， SDF里收到文件列表
	}
	//播放SD卡文件
	PlaySDFile(session,devID,sFile){
		const szMsg = 'SFF' + session + ':' + devID + ':' + sFile;
		this.sendMsg(szMsg);
		//返回值：
		//SFS:指令发送成功； SFN：参数错误或者设备异常
	}
	//停止播放SD文件
	StopSDFile(session,devID){
		const szMsg = 'SAF' + session + ':' + devID;
		this.sendMsg(szMsg);
		//返回值：
		//SAS:指令发送成功； SAN：参数错误或者设备异常
	}
	//拖拉播放进度条，设置播放位置
	SetSDFilePos(session,devID,nPos){
		const szMsg = 'SBF' + session + ':' + devID + ':' + nPos;
		this.sendMsg(szMsg);
		//返回值：
		//SBS:指令发送成功； SBN：参数错误或者设备异常
	}
	//sdfile.cgi?lens=0
	//lens:针对多摄摄像机，第几个镜头
	
	
	//透传CGI指令
	TransCGI(session,devID,sCGI){
		const szMsg = 'CGI' + session + ':' + devID + ':' + sCGI;
		this.sendMsg(szMsg);//停止监听
		//返回值
		//CGI：调用成功
		//CGN: 参数错误
		//CGR: 返回的cgi请求内容
	}
    //开始保存MP4; 
	//参数：
	//nLens:对于多摄摄像机的第几个镜头
    StartSaveMP4(session,devID,nLens){
        const szMsg = 'SAV' + session + ':' + devID + ':' + nLens;
		this.sendMsg(szMsg);  
        //返回值
        //SAA:参数错误
        //SAB:保存的路径
        //SAC:设备不存在或者没有连接
    }
    //停止保存MP4
	//参数：
	//nLens:对于多摄摄像机的第几个镜头
    StopSaveMP4(session,devID,nLens){
        const szMsg = 'SSA' + session + ':' + devID + ':' + nLens;
		this.sendMsg(szMsg);  
        //返回值
        //SSN:参数错误
        //SSB:成功
        //SSC:设备不存在或者没有连接        
    }
	//设置服务程序是否自动关闭。服务程序默认为自动关闭
	SetAutoClose(bAutoClose){
		const szMsg = 'AUC' + bAutoClose?'1':'0';
		this.sendMsg(szMsg);  
		//返回值
		//AUN:参数错误
		//AUS:成功
	}
	onSocketStatus(socketStatusCallback) {
		this.onSocketStatusCallback = socketStatusCallback
	}

	onCommand(commandCallback) {
		this.onCommandCallback = commandCallback
	}
	//http://localhost:813/session/devID/video.cgi?lens=0
	//lens:针对多摄摄像机，第几个镜头
	
	//http://localhost:813/session/devID/capture.jpg?lens=0
	//lens:针对多摄摄像机，第几个镜头
	
	

    InitSocket() {
        if (this.mReady) return true;
        this.mReady = false;
        let self = this;
        this.create(ExeHostIP, 3000, function () {
			if(self.onSocketStatus && !self.mReady)
				self.onSocketStatusCallback(true,'');
            self.mReady = true;
            // console.log('InitSocket:onConnected');
        }, function (szMsg) {
            self.mReady = true;
			if(self.onCommand)
				self.onCommandCallback(szMsg);
			// console.log('Socket:msg:'+szMsg);
        }, function (e, szErr) {
			if(self.onSocketStatus && self.mReady)
				self.onSocketStatusCallback(false,szErr);
            self.mReady = false;
            // console.log('InitSocket:error'+szErr);
        });
    }
}