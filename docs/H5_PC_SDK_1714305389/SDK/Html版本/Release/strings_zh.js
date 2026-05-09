var StrRes=[
	["IDS_CONNECT_SUCC","连接成功"],
	["IDS_ERR_CONNECT_DEV","视频连接失败"],
	["IDS_ERR_ACCESSAUTH","摄像机密码错误"],
	["IDS_ERR_EXTUSER","当前访问数超过最大数"],
	["IDS_ERR_ILLEGALDEV","非法设备"],
	["IDS_OFFLINE","不在线"],
	["IDS_ERR_FORBIDDEN","密码尝试次数超过限制，等待1小时"],
	["IDS_DUAL_AUTHING","正在鉴权"],
	["IDS_AUTH_TOKEN_ERR","鉴权失败"],
	["IDS_AUTH_SUCC","鉴权成功"],
	["IDS_RECEIVING","正常"],
	["IDS_PARAMERR","参数错误"],
	["IDS_LOGINFAIL_NOCONNECT","无法连接服务器."],
	["IDS_SERVER_ERR_PARAM","参数错误"],
	["IDS_SERVER_ERR_NOID","帐号不存在"],
	["IDS_ERRTYPE_PWD","密码错误"],
	["IDS_ConnectDev","连接中..."],
	["IDS_AlreadyExits","正常"],
	];
	
function getStr(sKey){
	for(var i=0;i<StrRes.length;i++){
		if(sKey == StrRes[i][0])
			return StrRes[i][1];
	}
	return "";
}