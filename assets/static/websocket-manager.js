/**
 * WebSocket 管理器
 * 用于Live2D界面的语音交流功能
 * @version 3.1.0 - 增强移动端音频解锁，多重策略确保兼容性
 */
class WebSocketManager {
    constructor() {
        this.ws = null;
        this.isConnecting = false;
        this.connectionStatus = 'disconnected';
        
        // 音频相关
        this.mediaRecorder = null;
        this.audioChunks = [];
        this.audioBlob = null;
        this.mediaStream = null;
        this.isRecording = false;
        this.audioAutoPlayEnabled = true;
        this.currentAudio = null;
        this.audioMessages = {};
        this.currentMessageId = null;
        this.isAutoPlaying = false;
        
        // 对话流程管理
        this.ignoredMessageIds = new Set(); // 需要忽略的消息ID集合
        this.lastUserInputTime = null; // 最后一次用户输入时间
        
        // 移动端音频播放相关
        this.isMobile = /iPhone|iPad|iPod|Android/i.test(navigator.userAgent);
        this.audioPlaybackUnlocked = false;
        
        // Howler.js 音频管理（必需）
        this.howlerSounds = new Map(); // 存储Howler音频对象
        this.unlockAttempts = 0; // 记录解锁尝试次数
        this.lastUnlockTime = 0; // 记录最后解锁时间
        
        // 强制要求Howler.js
        if (typeof Howl === 'undefined') {
            const error = new Error('Howler.js is required but not loaded. Please ensure Howler.js is included before WebSocketManager.');
            console.error('❌ 致命错误:', error.message);
            throw error;
        }
        
        console.log('✅ Howler.js 已加载，纯音频播放模式');
        
        // 检查Howler.js版本和可用功能
        console.log('🔍 Howler.js 可用方法:', {
            mute: typeof Howler.mute,
            volume: typeof Howler.volume,
            stop: typeof Howler.stop,
            state: typeof Howler.state,
            ctx: typeof Howler.ctx
        });
        
        // 设置全局配置
        try {
            // 检查音频上下文状态（如果可用）
            if (Howler.ctx && Howler.ctx.state) {
                console.log('🎵 音频上下文初始状态:', Howler.ctx.state);
                
                // 监听音频上下文状态变化
                if (Howler.ctx.onstatechange !== undefined) {
                    Howler.ctx.onstatechange = () => {
                        console.log('🔄 音频上下文状态变化:', Howler.ctx.state);
                        if (Howler.ctx.state === 'running') {
                            console.log('🔓 Howler.js: 音频上下文已运行，可能已解锁');
                            this.audioPlaybackUnlocked = true;
                            
                            // 触发自定义事件
                            if (typeof window !== 'undefined') {
                                window.dispatchEvent(new CustomEvent('audioUnlocked', {
                                    detail: { success: true, library: 'howler' }
                                }));
                            }
                        }
                    };
                }
            }
            
            // 设置iOS自动解锁（如果属性存在）
            if (typeof Howler.autoSuspend !== 'undefined') {
                Howler.autoSuspend = false;
                console.log('✅ 已禁用Howler.js自动挂起');
            }
            
            // 检查初始解锁状态
            this.checkInitialAudioState();
            
        } catch (error) {
            console.warn('⚠️ Howler.js 全局设置配置失败:', error);
        }
        
        // 输出版本信息
        console.log('🚀 WebSocket管理器版本: 3.1.4 - 优化对话流程：仅在发送消息时停止播放，而非录音开始时');
        console.log('⏰ 初始化时间:', new Date().toISOString());
        
        // VAD 相关
        this.vad = null;
        this.vadEnabled = false;
        this.vadListening = false;
        
        // 心跳相关
        this.pingInterval = null;
        this.lastPongTime = null;
        this.lastPingTime = null;
        
        // 默认配置
        this.config = {
            wsUrl: this.getWebSocketUrl(),
            audioFormat: 'webm',
            sampleRate: 48000,
            audioBitsPerSecond: 128000,
            autoConnect: false,
            enableVAD: true,
            enableHeartbeat: true,
            heartbeatInterval: 30000
        };
        

        
        // 事件回调
        this.callbacks = {
            onConnect: null,
            onDisconnect: null,
            onMessage: null,
            onError: null,
            onStatusChange: null,
            onAudioReceived: null,
            onRecordingStart: null,
            onRecordingStop: null
        };
    }
    
    // 获取或生成用户ID
    getUserId() {
        let uid = localStorage.getItem('ai_companion_uid');
        if (!uid) {
            // 生成一个随机的用户ID
            uid = 'uid_' + Math.random().toString(36).substr(2, 9) + '_' + Date.now().toString(36);
            localStorage.setItem('ai_companion_uid', uid);
            console.log('🆔 生成新的用户ID:', uid);
        } else {
            console.log('🆔 使用已保存的用户ID:', uid);
        }
        return uid;
    }
    
    // 增强移动端音频解锁方法（纯Howler.js + 多重策略）
    unlockAudioForMobile() {
        if (!this.audioPlaybackUnlocked) {
            const now = Date.now();
            
            // 避免频繁重复尝试（间隔至少1秒）
            if (now - this.lastUnlockTime < 1000) {
                console.log('⏰ 音频解锁尝试过于频繁，跳过');
                return;
            }
            
            this.lastUnlockTime = now;
            this.unlockAttempts++;
            
            console.log(`🔓 增强移动端音频解锁... (尝试 ${this.unlockAttempts})`);
            console.log('📱 设备信息:', {
                userAgent: navigator.userAgent.substring(0, 100),
                isMobile: this.isMobile,
                isHTTPS: location.protocol === 'https:',
                audioContext: Howler.ctx ? Howler.ctx.state : 'unavailable'
            });
            
            // 使用Howler.js进行多重策略音频解锁
            console.log('🎵 使用Howler.js进行音频解锁...');
            try {
                // 首先尝试通过Web Audio API解锁
                if (Howler.ctx && Howler.ctx.state === 'suspended') {
                    console.log('🔄 尝试恢复被挂起的音频上下文...');
                    Howler.ctx.resume().then(() => {
                        console.log('✅ 音频上下文已恢复');
                        this.audioPlaybackUnlocked = true;
                        
                        // 触发自定义事件
                        if (typeof window !== 'undefined') {
                            window.dispatchEvent(new CustomEvent('audioUnlocked', {
                                detail: { success: true, library: 'howler' }
                            }));
                        }
                    }).catch((error) => {
                        console.warn('⚠️ 音频上下文恢复失败:', error);
                        // 使用静音音频解锁
                        this.tryHowlerSilentAudio();
                    });
                } else {
                    // 如果上下文已经在运行或不存在，尝试播放静音音频
                    this.tryHowlerSilentAudio();
                }
                
                // 额外的移动端解锁策略
                this.tryMobileAudioUnlock();
                
            } catch (error) {
                console.error('❌ Howler.js 解锁异常:', error);
                console.error('❌ 音频解锁失败，无法播放音频');
            }
        }
    }
    
    // 移动端增强解锁策略
    tryMobileAudioUnlock() {
        console.log('📱 尝试移动端增强解锁策略...');
        
        // 策略1: 直接设置Howler全局音量
        try {
            Howler.volume(1.0);
            console.log('✅ Howler全局音量已设置');
        } catch (error) {
            console.warn('⚠️ 设置Howler全局音量失败:', error);
        }
        
        // 策略2: 检查并恢复音频上下文
        if (Howler.ctx) {
            console.log('🎵 音频上下文当前状态:', Howler.ctx.state);
            
            if (Howler.ctx.state === 'suspended') {
                // 再次尝试恢复
                Howler.ctx.resume().then(() => {
                    console.log('✅ 二次音频上下文恢复成功');
                    this.audioPlaybackUnlocked = true;
                }).catch((error) => {
                    console.warn('⚠️ 二次音频上下文恢复失败:', error);
                });
            }
        }
        
        // 策略3: 创建并播放极短的测试音频
        setTimeout(() => {
            this.tryUltraShortAudio();
        }, 500);
    }
    
    // 超短音频测试
    tryUltraShortAudio() {
        console.log('🎵 尝试超短音频测试...');
        
        try {
            const ultraShortSound = new Howl({
                src: ['data:audio/wav;base64,UklGRigAAABXQVZFZm10IBAAAAAAQIDqAgAAgQAAAEgAAAAQAUAA'],
                volume: 0.001, // 非常小的音量
                autoplay: false,
                preload: true,
                onload: () => {
                    console.log('🎵 超短音频加载成功');
                    const playPromise = ultraShortSound.play();
                    if (playPromise !== undefined) {
                        playPromise.catch((error) => {
                            console.warn('⚠️ 超短音频播放失败:', error);
                        });
                    }
                },
                onplay: () => {
                    console.log('✅ 超短音频播放成功，解锁状态更新');
                    this.audioPlaybackUnlocked = true;
                    
                    // 立即停止
                    setTimeout(() => {
                        ultraShortSound.stop();
                        ultraShortSound.unload();
                    }, 10);
                    
                    // 触发解锁事件
                    if (typeof window !== 'undefined') {
                        window.dispatchEvent(new CustomEvent('audioUnlocked', {
                            detail: { success: true, library: 'howler', method: 'ultra-short' }
                        }));
                    }
                },
                onplayerror: (id, error) => {
                    console.warn('⚠️ 超短音频播放错误:', error);
                    ultraShortSound.unload();
                }
            });
            
        } catch (error) {
            console.error('❌ 超短音频测试异常:', error);
        }
    }
    
    // 使用Howler.js播放静音音频进行解锁（所有平台）
    tryHowlerSilentAudio() {
        console.log('🔇 尝试使用Howler.js播放静音音频解锁...');
        
        try {
            // 创建一个静音测试音频
            const testSound = new Howl({
                src: ['data:audio/wav;base64,UklGRigAAABXQVZFZm10IBAAAAAAQIDqAgAAgQAAAEgAAAAQAUAA'],
                volume: 0.01,
                autoplay: false,
                onload: () => {
                    console.log('✅ Howler.js: 测试音频加载成功');
                    testSound.play();
                },
                onplay: () => {
                    console.log('✅ Howler.js: 音频已解锁（静音播放）');
                    this.audioPlaybackUnlocked = true;
                    testSound.stop();
                    testSound.unload();
                    
                    // 触发自定义事件
                    if (typeof window !== 'undefined') {
                        window.dispatchEvent(new CustomEvent('audioUnlocked', {
                            detail: { success: true, library: 'howler' }
                        }));
                    }
                },
                onplayerror: (id, error) => {
                    console.error('❌ Howler.js: 静音音频播放失败', error);
                    testSound.unload();
                    
                    // 设置解锁失败状态
                    this.audioPlaybackUnlocked = false;
                    if (typeof window !== 'undefined') {
                        window.dispatchEvent(new CustomEvent('audioUnlocked', {
                            detail: { success: false, error: error }
                        }));
                    }
                }
            });
            
            // 立即尝试播放
            testSound.play();
            
        } catch (error) {
            console.error('❌ Howler.js 静音音频播放异常:', error);
            // 设置解锁失败状态
            this.audioPlaybackUnlocked = false;
            if (typeof window !== 'undefined') {
                window.dispatchEvent(new CustomEvent('audioUnlocked', {
                    detail: { success: false, error: error }
                }));
            }
        }
    }
    

    
    // 检查初始音频解锁状态
    checkInitialAudioState() {
        
        try {
            // 检查Web Audio API上下文状态
            if (Howler.ctx) {
                const state = Howler.ctx.state;
                console.log('🔍 当前音频上下文状态:', state);
                
                if (state === 'running') {
                    console.log('✅ 音频上下文已在运行状态');
                    this.audioPlaybackUnlocked = true;
                } else if (state === 'suspended') {
                    console.log('⏸️ 音频上下文被挂起，需要用户交互解锁');
                    this.audioPlaybackUnlocked = false;
                } else {
                    console.log('❓ 音频上下文状态未知:', state);
                    this.audioPlaybackUnlocked = false;
                }
            } else {
                console.log('⚠️ 无法访问Howler音频上下文');
                // 降级：检查是否为移动端
                this.audioPlaybackUnlocked = !this.isMobile;
            }
            
            // 触发状态事件
            if (typeof window !== 'undefined') {
                window.dispatchEvent(new CustomEvent('audioStateChecked', {
                    detail: { 
                        unlocked: this.audioPlaybackUnlocked,
                        library: 'howler',
                        contextState: Howler.ctx ? Howler.ctx.state : 'unknown'
                    }
                }));
            }
            
        } catch (error) {
            console.warn('⚠️ 检查音频状态时出错:', error);
            this.audioPlaybackUnlocked = !this.isMobile; // 保守估计
        }
    }
    
    // 重新检查音频解锁状态
    recheckAudioUnlockStatus() {
        
        try {
            const previousState = this.audioPlaybackUnlocked;
            
            // 检查Web Audio API上下文状态
            if (Howler.ctx) {
                const contextState = Howler.ctx.state;
                console.log('🔍 重新检查音频上下文状态:', contextState);
                
                if (contextState === 'running') {
                    this.audioPlaybackUnlocked = true;
                } else if (contextState === 'suspended') {
                    this.audioPlaybackUnlocked = false;
                } else {
                    // 状态未知，保持现有状态
                    console.log('❓ 音频上下文状态未知:', contextState);
                }
            }
            
            // 如果状态发生变化，记录并通知
            if (previousState !== this.audioPlaybackUnlocked) {
                console.log(`🔄 音频解锁状态变化: ${previousState} → ${this.audioPlaybackUnlocked}`);
                
                if (this.audioPlaybackUnlocked) {
                    console.log('✅ 音频解锁状态已恢复');
                    if (typeof window !== 'undefined') {
                        window.dispatchEvent(new CustomEvent('audioUnlocked', {
                            detail: { success: true, library: 'howler', method: 'recheck' }
                        }));
                    }
                }
            }
            
            return this.audioPlaybackUnlocked;
            
        } catch (error) {
            console.warn('⚠️ 重新检查音频状态时出错:', error);
            return this.audioPlaybackUnlocked;
        }
    }
    
    // 获取适合当前环境的WebSocket URL
    getWebSocketUrl() {
        const isMobile = /iPhone|iPad|iPod|Android/i.test(navigator.userAgent);
        const isHTTPS = location.protocol === 'https:';
        const hostname = location.hostname;
        const port = location.port || (isHTTPS ? '443' : '80');
        const uid = this.getUserId();
        
        console.log('🌐 网络环境检测:', {
            isMobile,
            isHTTPS,
            hostname,
            protocol: location.protocol,
            host: location.host,
            uid: uid
        });
        
        const wsProtocol = isHTTPS ? 'wss:' : 'ws:';
        const wsUrl = `${wsProtocol}//${hostname}:${port}/ws?uid=${uid}`;
        console.log('🖥️ 桌面端使用默认连接:', wsUrl);
        return wsUrl;
    }
    
    // 尝试获取本机IP地址（简单方法）
    getLocalIP() {
        // 这里可以返回预设的开发服务器IP，或者让用户配置
        // 在实际部署时，这应该是服务器的实际IP地址
        
        // 检查是否有预设的服务器IP
        if (window.AI_COMPANION_SERVER_IP) {
            return window.AI_COMPANION_SERVER_IP;
        }
        
        // 尝试从URL参数获取服务器IP
        const urlParams = new URLSearchParams(window.location.search);
        const serverIP = urlParams.get('server') || urlParams.get('ip');
        if (serverIP) {
            console.log('📋 从URL参数获取服务器IP:', serverIP);
            return serverIP;
        }
        
        // 常见的开发环境IP地址（用户需要根据实际情况修改）
        const commonIPs = [
            '192.168.1.100',  // 常见的路由器分配IP段
            '192.168.0.100',
            '10.0.0.100'
        ];
        
        console.log('💡 提示：如果连接失败，请尝试以下方法之一：');
        console.log('  1. 在URL中添加参数: ?server=您的电脑IP地址');
        console.log('  2. 设置 window.AI_COMPANION_SERVER_IP = "您的电脑IP地址"');
        console.log('  3. 确保电脑和手机在同一WiFi网络中');
        console.log('  4. 常见IP地址:', commonIPs);
        
        return null;
    }
    
    // 注册事件回调
    on(event, callback) {
        if (this.callbacks.hasOwnProperty('on' + event.charAt(0).toUpperCase() + event.slice(1))) {
            this.callbacks['on' + event.charAt(0).toUpperCase() + event.slice(1)] = callback;
        }
    }
    
    // 触发事件
    emit(event, ...args) {
        const callbackName = 'on' + event.charAt(0).toUpperCase() + event.slice(1);
        if (this.callbacks[callbackName]) {
            this.callbacks[callbackName](...args);
        }
    }
    
    // 连接 WebSocket
    async connect(url = null) {
        if (this.isConnecting || (this.ws && this.ws.readyState === WebSocket.OPEN)) {
            return false;
        }
        
        const wsUrl = url || this.config.wsUrl;
        this.isConnecting = true;
        this.updateStatus('connecting');
        
        try {
            this.ws = new WebSocket(wsUrl);
            
            this.ws.onopen = (event) => {
                this.isConnecting = false;
                this.updateStatus('connected');
                this.emit('connect', event);
                this.initMicrophone();
                if (this.config.enableHeartbeat) {
                    this.startHeartbeat();
                }
            };
            
            this.ws.onmessage = (event) => {
                this.handleMessage(event);
            };
            
            this.ws.onclose = (event) => {
                this.isConnecting = false;
                this.updateStatus('disconnected');
                this.stopHeartbeat();
                this.emit('disconnect', event);
            };
            
            this.ws.onerror = (error) => {
                this.isConnecting = false;
                this.updateStatus('error');
                this.emit('error', error);
            };
            
            return true;
        } catch (error) {
            this.isConnecting = false;
            this.updateStatus('error');
            this.emit('error', error);
            return false;
        }
    }
    
    // 断开连接
    disconnect() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        this.stopHeartbeat();
        this.stopVAD();
        this.stopRecording();
    }
    
    // 更新连接状态
    updateStatus(status) {
        this.connectionStatus = status;
        this.emit('statusChange', status);
    }
    
    // 处理接收到的消息
    handleMessage(event) {
        try {
            const messageData = JSON.parse(event.data);
            
            if (messageData.type === 'text') {
                this.emit('message', {
                    type: 'text',
                    content: messageData.data,
                    timestamp: Date.now()
                });
                
            } else if (messageData.type === 'audio') {
                this.emit('audioReceived', messageData);
                if (this.audioAutoPlayEnabled) {
                    this.autoPlayAudio(messageData);
                }
                
            } else if (messageData.type === 'pong') {
                this.handlePongMessage(messageData);
            }
            
            this.emit('message', {
                type: messageData.type,
                data: messageData,
                timestamp: Date.now()
            });
            
        } catch (error) {
            // 非JSON消息
            this.emit('message', {
                type: 'raw',
                content: event.data,
                timestamp: Date.now()
            });
        }
    }
    
    // 发送文本消息
    sendText(text) {
        if (!this.isConnected()) {
            this.emit('error', new Error('WebSocket未连接'));
            return false;
        }
        
        try {
            const message = {
                type: 'text',
                data: text
            };
            
            // 发送新消息时管理对话流程
            this.handleUserInput('text');
            
            this.ws.send(JSON.stringify(message));
            
            return true;
        } catch (error) {
            this.emit('error', error);
            return false;
        }
    }
    
    // 发送音频
    async sendAudio() {
        if (!this.audioBlob || !this.isConnected()) {
            return false;
        }
        
        try {
            const reader = new FileReader();
            reader.onload = () => {
                const base64Data = reader.result.split(',')[1];
                const audioMessage = {
                    type: 'audio',
                    format: 'webm',
                    size: this.audioBlob.size,
                    data: base64Data
                };
                
                // 发送新音频时管理对话流程
                this.handleUserInput('audio');
                
                this.ws.send(JSON.stringify(audioMessage));
                
                // 清空录音
                this.audioBlob = null;
            };
            reader.readAsDataURL(this.audioBlob);
            return true;
        } catch (error) {
            this.emit('error', error);
            return false;
        }
    }
    
    // 检查连接状态
    isConnected() {
        return this.ws && this.ws.readyState === WebSocket.OPEN;
    }
    
    // 初始化麦克风
    async initMicrophone() {
        console.log('🎤 开始初始化麦克风...');
        console.log('📱 设备信息:', {
            userAgent: navigator.userAgent,
            platform: navigator.platform,
            isMobile: /iPhone|iPad|iPod|Android/i.test(navigator.userAgent)
        });
        
        // 检查媒体设备API支持
        if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
            console.error('❌ 浏览器不支持媒体设备API');
            this.emit('error', new Error('您的浏览器不支持麦克风访问，请使用现代浏览器或确保使用HTTPS连接'));
            return false;
        }
        
        // 检查HTTPS环境（移动端要求）
        const isMobile = /iPhone|iPad|iPod|Android/i.test(navigator.userAgent);
        const isHTTPS = location.protocol === 'https:';
        
        if (isMobile && !isHTTPS) {
            console.error('❌ 移动端需要HTTPS环境才能访问麦克风');
            this.emit('error', new Error('移动端需要HTTPS环境才能访问麦克风，请使用HTTPS连接或配置本地HTTPS服务器'));
            return false;
        }
        
        try {
            // 移动端优化的音频约束
            const audioConstraints = {
                echoCancellation: true,
                noiseSuppression: true,
                autoGainControl: true
            };
            
            // 桌面端才设置采样率，移动端使用默认值
            if (!isMobile) {
                audioConstraints.sampleRate = this.config.sampleRate;
            }
            
            console.log('🔑 请求麦克风权限, 音频约束:', audioConstraints);
            console.log('📱 移动端设备:', isMobile ? '是' : '否');
            console.log('🔒 HTTPS环境:', isHTTPS ? '是' : '否');
            
            this.mediaStream = await navigator.mediaDevices.getUserMedia({
                audio: audioConstraints
            });
            
            console.log('✅ 麦克风权限获取成功, mediaStream:', !!this.mediaStream);
            
            // 移动端额外检查
            if (this.mediaStream && this.mediaStream.getAudioTracks) {
                const audioTracks = this.mediaStream.getAudioTracks();
                console.log('🎵 音频轨道数量:', audioTracks.length);
                if (audioTracks.length > 0) {
                    console.log('🎚️ 音频轨道设置:', audioTracks[0].getSettings());
                }
            }
            

            
            if (this.config.enableVAD) {
                console.log('🔊 VAD已启用，开始初始化VAD...');
                await this.initVAD();
            } else {
                console.log('❌ VAD未启用');
                // 即使VAD未启用，也要触发事件告知外部
                this.emit('vadReady', false);
            }
            
            return true;
        } catch (error) {
            console.error('❌ 麦克风初始化失败:', error);
            
            // 移动端友好的错误提示
            if (error.name === 'NotAllowedError') {
                console.error('🚫 用户拒绝了麦克风权限');
                this.emit('error', new Error('请允许麦克风访问权限，语音功能需要使用麦克风'));
            } else if (error.name === 'NotFoundError') {
                console.error('🔍 未找到麦克风设备');
                this.emit('error', new Error('未找到可用的麦克风设备'));
            } else if (error.name === 'NotReadableError') {
                console.error('🔒 麦克风被其他应用占用');
                this.emit('error', new Error('麦克风正被其他应用使用，请关闭其他应用后重试'));
            } else if (error.name === 'OverconstrainedError') {
                console.error('⚙️ 音频约束不被支持');
                // 移动端降级处理：使用最基本的约束重试
                try {
                    console.log('🔄 尝试使用基础音频约束重试...');
                    this.mediaStream = await navigator.mediaDevices.getUserMedia({
                        audio: true
                    });
                    console.log('✅ 基础约束重试成功');
                    
                    if (this.config.enableVAD) {
                        await this.initVAD();
                    }
                    return true;
                } catch (retryError) {
                    console.error('❌ 基础约束重试也失败:', retryError);
                    this.emit('error', new Error('音频设备不兼容当前配置'));
                }
            } else if (error.message && error.message.includes('getUserMedia')) {
                console.error('🔒 媒体设备API不可用');
                this.emit('error', new Error('媒体设备API不可用，请确保使用HTTPS连接或现代浏览器'));
            } else {
                this.emit('error', error);
            }
            return false;
        }
    }
    
    // 初始化语音激活检测
    async initVAD() {
        try {
            if (!window.vad) {
                console.warn('VAD库未加载，跳过语音检测初始化');
                return;
            }
            
            this.vad = await window.vad.MicVAD.new({
                onSpeechStart: () => {
                    if (this.vadListening && !this.isRecording) {
                        this.startRecording();
                    }
                },
                onSpeechEnd: () => {
                    if (this.vadListening && this.isRecording) {
                        this.stopRecording();
                        setTimeout(() => {
                            if (this.audioBlob) {
                                this.sendAudio();
                            }
                        }, 100);
                    }
                },
                positiveSpeechThreshold: 0.5,
                negativeSpeechThreshold: 0.35,
                redemptionFrames: 8,
                workletURL: 'https://cdn.jsdelivr.net/npm/@ricky0123/vad-web@0.0.22/dist/vad.worklet.bundle.min.js',
                modelURL: 'https://cdn.jsdelivr.net/npm/@ricky0123/vad-web@0.0.22/dist/silero_vad.onnx'
            });
            
            this.vadEnabled = true;
            console.log('✅ VAD初始化成功');
            this.emit('vadReady', true);
        } catch (error) {
            console.warn('❌ VAD初始化失败:', error);
            this.vadEnabled = false;
            this.emit('vadReady', false);
        }
    }
    
    // 启动/停止语音监听
    toggleVAD() {
        if (!this.vad) return false;
        
        if (this.vadListening) {
            this.vad.pause();
            this.vadListening = false;
        } else {
            this.vad.start();
            this.vadListening = true;
        }
        
        return this.vadListening;
    }
    
    // 停止VAD
    stopVAD() {
        if (this.vad && this.vadListening) {
            this.vad.pause();
            this.vadListening = false;
        }
    }
    
    // 开始录音
    startRecording() {
        if (!this.mediaStream || this.isRecording) return false;
        
        this.audioChunks = [];
        
        try {
            this.mediaRecorder = new MediaRecorder(this.mediaStream, {
                mimeType: 'audio/webm;codecs=opus',
                audioBitsPerSecond: this.config.audioBitsPerSecond
            });
            
            this.mediaRecorder.ondataavailable = (event) => {
                if (event.data.size > 0) {
                    this.audioChunks.push(event.data);
                }
            };
            
            this.mediaRecorder.onstop = () => {
                this.audioBlob = new Blob(this.audioChunks, { type: 'audio/webm' });
                this.emit('recordingStop', this.audioBlob);
            };
            
            this.mediaRecorder.start();
            this.isRecording = true;
            this.emit('recordingStart');
            
            return true;
        } catch (error) {
            this.emit('error', error);
            return false;
        }
    }
    
    // 停止录音
    stopRecording() {
        if (!this.mediaRecorder || !this.isRecording) return false;
        
        this.mediaRecorder.stop();
        this.isRecording = false;
        return true;
    }
    
    // 手动录音切换
    toggleRecording() {
        if (this.isRecording) {
            return this.stopRecording();
        } else {
            return this.startRecording();
        }
    }
    
    // 自动播放音频
    autoPlayAudio(audioData) {
        const messageId = audioData.message_id || 'default';
        
        // 检查是否应该忽略此消息的音频片段
        if (this.ignoredMessageIds.has(messageId)) {
            console.log(`🚫 忽略已取消消息 ${messageId} 的音频片段 (chunk_id: ${audioData.chunk_id || 0})`);
            return;
        }
        
        const audioItem = {
            data: audioData.data,
            format: audioData.format || 'webm',
            chunkId: audioData.chunk_id || 0,
            isEnd: audioData.is_end || false,
            size: audioData.size || 0,
            messageId: messageId
        };
        
        // 检查自动播放是否被禁用
        if (!this.audioAutoPlayEnabled) {
            return;
        }
        
        // 如果是新消息，清除所有旧消息的音频并重新开始
        if (this.currentMessageId && this.currentMessageId !== messageId) {
            console.log(`🔄 检测到新的对话轮次 ${messageId}，清除旧轮次 ${this.currentMessageId} 的所有音频`);
            this.clearPreviousMessageAudio(this.currentMessageId);
        }
        
        // 按消息ID分组存储
        if (!this.audioMessages[messageId]) {
            this.audioMessages[messageId] = [];
        }
        
        this.audioMessages[messageId].push(audioItem);
        
        // 开始播放新消息
        if (!this.isAutoPlaying || this.currentMessageId !== messageId) {
            this.currentMessageId = messageId;
            this.playAudioMessage(messageId);
        }
    }
    
    // 播放音频消息（Howler.js增强版）
    playAudioMessage(messageId) {
        if (!this.audioMessages[messageId] || this.audioMessages[messageId].length === 0) {
            return;
        }
        
        // 检查该消息是否被忽略
        if (this.isMessageIgnored(messageId)) {
            console.log(`🚫 消息 ${messageId} 在忽略列表中，跳过播放`);
            // 清理被忽略消息的数据
            this.clearMessageAudio(messageId);
            return;
        }
        
        // 检查音频自动播放是否启用
        if (!this.audioAutoPlayEnabled) {
            console.log('🔇 音频自动播放已禁用，跳过播放');
            return;
        }
        
        // 实时检查音频解锁状态
        this.recheckAudioUnlockStatus();
        
        // 检查音频解锁状态
        if (!this.audioPlaybackUnlocked) {
            console.log('🎵 音频未解锁，尝试重新解锁...');
            
            // 尝试自动重新解锁
            this.unlockAudioForMobile();
            
            // 短暂延迟后重新检查
            setTimeout(() => {
                if (!this.audioPlaybackUnlocked) {
                    console.log('⚠️ 自动重新解锁失败，显示用户提示');
                    if (typeof window !== 'undefined') {
                        window.dispatchEvent(new CustomEvent('audioPlaybackBlocked', {
                            detail: { 
                                reason: 'unlock_required',
                                messageId: messageId,
                                attempts: this.unlockAttempts || 0
                            }
                        }));
                    }
                } else {
                    console.log('✅ 重新解锁成功，继续播放音频');
                    // 递归调用以继续播放
                    this.playAudioMessage(messageId);
                }
            }, 200);
            
            return;
        }
        
        this.isAutoPlaying = true;
        this.currentMessageId = messageId;
        
        const sortedChunks = this.audioMessages[messageId].sort((a, b) => a.chunkId - b.chunkId);
        
        // 使用 Howler.js 播放音频（纯模式）
        console.log('🎵 使用 Howler.js 播放音频 (纯模式)');
        this.playAudioWithHowler(messageId, sortedChunks);
    }
    
    // 使用Howler.js播放音频
    playAudioWithHowler(messageId, sortedChunks) {
        console.log('🎵 使用Howler.js播放音频, chunks:', sortedChunks.length);
        let currentIndex = 0;
        
        const playNextChunk = () => {
            if (currentIndex >= sortedChunks.length) {
                this.isAutoPlaying = false;
                this.currentAudio = null;
                delete this.audioMessages[messageId];
                
                // 清理Howler音频对象
                if (this.howlerSounds.has(messageId)) {
                    const sound = this.howlerSounds.get(messageId);
                    sound.unload();
                    this.howlerSounds.delete(messageId);
                }
                
                // 检查是否有其他待播放消息
                const pendingMessages = Object.keys(this.audioMessages);
                if (pendingMessages.length > 0) {
                    this.playAudioMessage(pendingMessages[0]);
                } else {
                    this.currentMessageId = null;
                }
                return;
            }
            
            const audioItem = sortedChunks[currentIndex];
            currentIndex++;
            
            try {
                const binaryString = atob(audioItem.data);
                const bytes = new Uint8Array(binaryString.length);
                for (let i = 0; i < binaryString.length; i++) {
                    bytes[i] = binaryString.charCodeAt(i);
                }
                
                const audioBlob = new Blob([bytes], { type: `audio/${audioItem.format}` });
                const audioUrl = URL.createObjectURL(audioBlob);
                
                // 创建Howler音频对象
                const sound = new Howl({
                    src: [audioUrl],
                    format: ['webm', 'mp3', 'wav'], // 支持多种格式
                    volume: 1.0,
                    preload: true,
                    autoplay: false,
                    onload: () => {
                        console.log('🎵 Howler.js: 音频chunk加载成功', currentIndex - 1);
                        sound.play();
                    },
                    onplay: () => {
                        console.log('🔊 Howler.js: 音频chunk播放开始', currentIndex - 1);
                        this.currentAudio = sound;
                    },
                    onend: () => {
                        console.log('✅ Howler.js: 音频chunk播放结束', currentIndex - 1);
                        URL.revokeObjectURL(audioUrl);
                        sound.unload();
                        setTimeout(playNextChunk, 50);
                    },
                    onloaderror: (id, error) => {
                        console.error('❌ Howler.js: 音频加载错误', id, error);
                        URL.revokeObjectURL(audioUrl);
                        sound.unload();
                        
                        // 停止播放，设置错误状态
                        this.isAutoPlaying = false;
                        this.currentAudio = null;
                        console.error('❌ 音频播放失败，无法继续');
                    },
                    onplayerror: (id, error) => {
                        console.error('❌ Howler.js: 音频播放错误', id, error);
                        URL.revokeObjectURL(audioUrl);
                        sound.unload();
                        
                        // 显示用户交互提示
                        if (typeof window !== 'undefined') {
                            window.dispatchEvent(new CustomEvent('audioPlaybackBlocked', {
                                detail: { 
                                    reason: 'user_interaction_required',
                                    messageId: messageId,
                                    error: error
                                }
                            }));
                        }
                        
                        // 停止播放
                        this.isAutoPlaying = false;
                        this.currentAudio = null;
                    },
                    onstop: () => {
                        URL.revokeObjectURL(audioUrl);
                        sound.unload();
                    }
                });
                
                // 存储音频对象以便管理
                this.howlerSounds.set(`${messageId}_${currentIndex}`, sound);
                
            } catch (error) {
                console.error('❌ Howler.js: 音频处理错误', error);
                // 停止播放
                this.isAutoPlaying = false;
                this.currentAudio = null;
            }
        };
        
        playNextChunk();
    }
    

    
    // 打断当前音频
    interruptCurrentAudio() {
        if (this.currentAudio) {
            // Howler音频对象
            this.currentAudio.stop();
            this.currentAudio.unload();
            this.currentAudio = null;
        }
        this.isAutoPlaying = false;
        this.currentMessageId = null;
    }
    
    // 清除指定消息的所有音频
    clearMessageAudio(messageId) {
        if (!messageId) return;
        
        console.log(`🗑️ 清除消息 ${messageId} 的所有音频数据`);
        
        // 停止并清理该消息的所有Howler音频对象
        this.howlerSounds.forEach((sound, key) => {
            if (key.startsWith(messageId)) {
                try {
                    sound.stop();
                    sound.unload();
                    this.howlerSounds.delete(key);
                } catch (error) {
                    console.warn(`⚠️ 清理音频对象 ${key} 时出错:`, error);
                }
            }
        });
        
        // 删除消息数据
        if (this.audioMessages[messageId]) {
            delete this.audioMessages[messageId];
        }
        
        // 如果正在播放这个消息，停止播放
        if (this.currentMessageId === messageId) {
            this.isAutoPlaying = false;
            this.currentAudio = null;
            this.currentMessageId = null;
        }
    }
    
    // 清除前一个消息的音频响应（保留原方法以兼容）
    clearPreviousMessageAudio(previousMessageId) {
        this.clearMessageAudio(previousMessageId);
    }
    
    // 停止所有音频播放
    stopAllAudio() {
        this.interruptCurrentAudio();
        
        // 停止所有Howler音频对象
        if (this.howlerSounds.size > 0) {
            console.log('🛑 停止所有Howler音频对象:', this.howlerSounds.size);
            this.howlerSounds.forEach((sound, key) => {
                try {
                    sound.stop();
                    sound.unload();
                } catch (error) {
                    console.warn('⚠️ 停止Howler音频出错:', key, error);
                }
            });
            this.howlerSounds.clear();
        }
        
        // 全局停止Howler音频
        if (typeof Howler !== 'undefined') {
            Howler.stop();
        }
        
        this.audioMessages = {};
    }
    
    // 清除所有音频响应
    clearAllAudioResponses() {
        console.log('🗑️ 清除所有音频响应');
        
        // 停止当前播放
        this.interruptCurrentAudio();
        
        // 停止所有Howler音频对象
        if (this.howlerSounds.size > 0) {
            this.howlerSounds.forEach((sound, key) => {
                try {
                    sound.stop();
                    sound.unload();
                } catch (error) {
                    console.warn('⚠️ 停止Howler音频出错:', key, error);
                }
            });
            this.howlerSounds.clear();
        }
        
        // 全局停止Howler音频
        if (typeof Howler !== 'undefined') {
            Howler.stop();
        }
        
        // 清空所有消息存储
        this.audioMessages = {};
        this.currentMessageId = null;
        this.isAutoPlaying = false;
        this.currentAudio = null;
        
        // 注意：不清理忽略列表，因为服务器可能还会发送被忽略消息的音频片段
        // 忽略列表将通过定时清理机制自动清理
    }
    
    // 处理用户新输入时的对话流程管理（仅在发送消息时触发）
    handleUserInput(inputType = 'unknown') {
        console.log(`🔄 用户发送消息检测 (${inputType})，管理对话流程`);
        
        // 记录用户输入时间
        this.lastUserInputTime = Date.now();
        
        // 将当前正在播放的消息ID加入忽略列表
        if (this.currentMessageId) {
            this.ignoredMessageIds.add(this.currentMessageId);
            console.log(`📋 将消息 ${this.currentMessageId} 加入忽略列表，后续音频片段将被丢弃`);
        }
        
        // 将所有缓存的消息ID也加入忽略列表
        Object.keys(this.audioMessages).forEach(messageId => {
            if (messageId !== this.currentMessageId) {
                this.ignoredMessageIds.add(messageId);
                console.log(`📋 将缓存消息 ${messageId} 加入忽略列表`);
            }
        });
        
        // 立即停止所有当前播放的音频
        this.clearAllAudioResponses();
        
        // 触发对话流程变更事件
        this.emit('conversationFlowChanged', {
            type: 'user_message_sent',
            inputType: inputType,
            timestamp: this.lastUserInputTime,
            previousMessageId: this.currentMessageId,
            ignoredMessageIds: Array.from(this.ignoredMessageIds)
        });
        
        // 重置状态，准备接收新的响应
        this.currentMessageId = null;
        this.isAutoPlaying = false;
        
        // 定时清理忽略列表（避免内存泄露）
        this.scheduleIgnoreListCleanup();
    }
    
    // 定时清理忽略列表
    scheduleIgnoreListCleanup() {
        // 30秒后清理忽略列表
        setTimeout(() => {
            const cleanupTime = Date.now();
            const oldSize = this.ignoredMessageIds.size;
            
            // 清理超过30秒的忽略项（基于用户输入时间）
            if (this.lastUserInputTime && (cleanupTime - this.lastUserInputTime) > 30000) {
                this.ignoredMessageIds.clear();
                if (oldSize > 0) {
                    console.log(`🧹 清理忽略列表：移除 ${oldSize} 个旧消息ID`);
                }
            }
        }, 30000);
    }
    
    // 手动清理忽略列表
    clearIgnoreList() {
        const oldSize = this.ignoredMessageIds.size;
        this.ignoredMessageIds.clear();
        if (oldSize > 0) {
            console.log(`🧹 手动清理忽略列表：移除 ${oldSize} 个消息ID`);
        }
    }
    
    // 检查消息是否被忽略
    isMessageIgnored(messageId) {
        return this.ignoredMessageIds.has(messageId);
    }
    
    // 心跳机制
    startHeartbeat() {
        this.pingInterval = setInterval(() => {
            if (this.isConnected()) {
                this.lastPingTime = Date.now();
                const pingMessage = {
                    type: 'ping',
                    timestamp: this.lastPingTime
                };
                this.ws.send(JSON.stringify(pingMessage));
                this.checkConnectionHealth();
            } else {
                this.stopHeartbeat();
            }
        }, this.config.heartbeatInterval);
    }
    
    stopHeartbeat() {
        if (this.pingInterval) {
            clearInterval(this.pingInterval);
            this.pingInterval = null;
            this.lastPongTime = null;
            this.lastPingTime = null;
        }
    }
    
    checkConnectionHealth() {
        const now = Date.now();
        if (this.lastPongTime && (now - this.lastPongTime) > 60000) {
            this.updateStatus('unstable');
        }
    }
    
    handlePongMessage(pongData) {
        this.lastPongTime = Date.now();
        if (this.connectionStatus !== 'connected') {
            this.updateStatus('connected');
        }
    }
    
    // 获取状态
    getStatus() {
        return {
            version: '3.1.4',
            lastUpdate: '优化对话流程：仅在发送消息时停止播放，而非录音开始时',
            connectionStatus: this.connectionStatus,
            isConnected: this.isConnected(),
            isRecording: this.isRecording,
            vadListening: this.vadListening,
            vadEnabled: this.vadEnabled,
            audioAutoPlayEnabled: this.audioAutoPlayEnabled,
            isAutoPlaying: this.isAutoPlaying,
            audioPlaybackUnlocked: this.audioPlaybackUnlocked,
            howlerSoundsCount: this.howlerSounds.size,
            isMobile: this.isMobile,
            unlockAttempts: this.unlockAttempts,
            lastUnlockTime: this.lastUnlockTime,
            audioPlayMode: 'Howler.js (纯模式)',
            currentMessageId: this.currentMessageId,
            audioMessagesCount: Object.keys(this.audioMessages).length,
            conversationFlowOptimized: true,
            userInteractionResponseEnabled: true,
            ignoredMessageIdsCount: this.ignoredMessageIds.size,
            lastUserInputTime: this.lastUserInputTime,
            audioFragmentIgnoreEnabled: true,
            stopOnMessageSendOnly: true
        };
    }
    
    // 获取版本信息
    getVersion() {
        return {
            version: '3.1.4',
            description: '优化对话流程：仅在发送消息时停止播放，而非录音开始时',
            timestamp: new Date().toISOString(),
            majorChanges: [
                '🎵 完全移除原生Audio API支持',
                '🛠️ 强制要求Howler.js，加载失败时直接抛出错误',
                '🔧 简化代码架构，移除所有降级逻辑',
                '⚡ 提升性能，减少代码复杂度',
                '🎯 专注于Howler.js的最佳实践',
                '📱 增强移动端音频解锁机制',
                '🔓 多重音频解锁策略（上下文恢复、静音播放、超短音频）',
                '👆 增强用户交互检测和音频解锁',
                '📋 改进移动端音频故障提示和解决方案',
                '🔄 优化对话流优先级：新输入时立即清除旧响应',
                '🗑️ 智能音频队列管理：按消息ID分组清理',
                '⚡ 减少音频播放延迟和冲突',
                '🎤 完善用户交互响应：录音、VAD、手动操作统一管理',
                '📞 实时对话流程管理：任何用户输入都立即停止AI播放',
                '🚫 音频片段忽略机制：彻底阻止服务器继续发送的旧消息音频',
                '🧹 自动内存清理：定时清理忽略列表避免内存泄露',
                '🎯 精准停止时机：仅在发送消息时停止播放，允许用户边听边准备'
            ],
            removedFeatures: [
                '原生Audio播放方法',
                '音频播放降级策略',
                'isHowlerAvailable兼容检查',
                '原生音频解锁方法',
                '平台差异化处理逻辑',
                '录音开始时停止播放（现在仅在发送消息时停止）'
            ],
            preservedFeatures: [
                '智能音频解锁重试机制',
                '实时音频状态检查',
                '手动解锁按钮界面',
                '防止频繁重复尝试',
                '增强的错误处理和用户提示'
            ],
            newFeatures: [
                '对话流优先级管理：新对话轮次自动清除旧响应',
                '按消息ID分组的音频队列管理',
                '智能音频清理：clearMessageAudio() 方法',
                '完整的音频响应清除：clearAllAudioResponses() 方法',
                '实时对话状态跟踪：currentMessageId 和 audioMessagesCount',
                '优化的用户体验：减少音频播放冲突和延迟',
                '统一的用户交互管理：handleUserInput() 方法',
                '多种输入类型检测：文本、语音消息',
                '对话流程变更事件：conversationFlowChanged 事件',
                '精准响应机制：仅消息发送时停止AI音频播放',
                '音频片段忽略机制：ignoredMessageIds 集合管理',
                '智能忽略检查：isMessageIgnored() 方法',
                '自动清理机制：scheduleIgnoreListCleanup() 和 clearIgnoreList()',
                '彻底阻止旧音频：服务器继续发送的音频片段被完全丢弃',
                '内存优化：定时清理避免忽略列表无限增长',
                '用户友好：录音时不打断播放，仅发送时才打断'
            ]
        };
    }
    
    // 设置配置
    setConfig(newConfig) {
        this.config = { ...this.config, ...newConfig };
    }
    
    // 销毁实例
    destroy() {
        this.disconnect();
        if (this.mediaStream) {
            this.mediaStream.getTracks().forEach(track => track.stop());
        }
        
        // 清理忽略列表
        this.clearIgnoreList();
    }
    
    // 调试方法：获取忽略列表信息
    getIgnoreListInfo() {
        return {
            ignoredMessageIds: Array.from(this.ignoredMessageIds),
            ignoredCount: this.ignoredMessageIds.size,
            lastUserInputTime: this.lastUserInputTime,
            timeSinceLastInput: this.lastUserInputTime ? Date.now() - this.lastUserInputTime : null
        };
    }
    
    // 调试方法：模拟服务器发送旧消息音频
    simulateOldAudioFragment(messageId, chunkId = 0) {
        console.log(`🧪 模拟服务器发送旧消息 ${messageId} 的音频片段 ${chunkId}`);
        this.autoPlayAudio({
            message_id: messageId,
            chunk_id: chunkId,
            data: 'fake_audio_data',
            format: 'webm',
            is_end: false,
            size: 1024
        });
    }
}

// 导出
window.WebSocketManager = WebSocketManager; 