/**
 * WebSocket 管理器
 * 用于Live2D界面的语音交流功能
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
        
        // 移动端音频播放相关
        this.isMobile = /iPhone|iPad|iPod|Android/i.test(navigator.userAgent);
        this.audioPlaybackUnlocked = false;
        
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
    
    // 简单的音频解锁方法
    unlockAudioForMobile() {
        if (this.isMobile && !this.audioPlaybackUnlocked) {
            console.log('📱 移动端音频解锁...');
            try {
                // 创建并立即播放静音音频
                const silentAudio = new Audio();
                silentAudio.src = 'data:audio/wav;base64,UklGRigAAABXQVZFZm10IBAAAAAAQIDqAgAAgQAAAEgAAAAQAUAA';
                silentAudio.volume = 0.01;
                silentAudio.play().then(() => {
                    console.log('✅ 移动端音频已解锁');
                    this.audioPlaybackUnlocked = true;
                }).catch(() => {
                    console.log('⚠️ 静音音频播放失败，但标记为已解锁');
                    this.audioPlaybackUnlocked = true;
                });
            } catch (error) {
                console.log('⚠️ 音频解锁异常，但标记为已解锁');
                this.audioPlaybackUnlocked = true;
            }
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
            this.ws.send(JSON.stringify(message));
            
            // 发送消息时打断当前播放
            this.interruptCurrentAudio();
            
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
                
                this.ws.send(JSON.stringify(audioMessage));
                
                // 发送音频时打断当前播放
                this.interruptCurrentAudio();
                
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
        
        // 如果是新消息，打断当前播放
        if (this.currentMessageId && this.currentMessageId !== messageId) {
            this.interruptCurrentAudio();
        }
        
        // 按消息ID分组存储
        if (!this.audioMessages[messageId]) {
            this.audioMessages[messageId] = [];
        }
        
        this.audioMessages[messageId].push(audioItem);
        
        // 开始播放
        if (!this.isAutoPlaying || this.currentMessageId !== messageId) {
            this.currentMessageId = messageId;
            this.playAudioMessage(messageId);
        }
    }
    
    // 播放音频消息
    playAudioMessage(messageId) {
        if (!this.audioMessages[messageId] || this.audioMessages[messageId].length === 0) {
            return;
        }
        
        this.isAutoPlaying = true;
        this.currentMessageId = messageId;
        
        const sortedChunks = this.audioMessages[messageId].sort((a, b) => a.chunkId - b.chunkId);
        let currentIndex = 0;
        
        const playNextChunk = () => {
            if (currentIndex >= sortedChunks.length) {
                this.isAutoPlaying = false;
                this.currentAudio = null;
                delete this.audioMessages[messageId];
                
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
                this.currentAudio = new Audio(audioUrl);
                
                this.currentAudio.oncanplay = () => {
                    this.currentAudio.play().catch(error => {
                        console.error('播放失败:', error);
                        // 移动端特殊处理：如果播放失败且未解锁，尝试解锁
                        if (this.isMobile && !this.audioPlaybackUnlocked) {
                            this.unlockAudioForMobile();
                        }
                        playNextChunk();
                    });
                };
                
                this.currentAudio.onended = () => {
                    URL.revokeObjectURL(audioUrl);
                    setTimeout(playNextChunk, 50);
                };
                
                this.currentAudio.onerror = (error) => {
                    console.error('🔊 音频加载错误:', error);
                    URL.revokeObjectURL(audioUrl);
                    playNextChunk();
                };
                
            } catch (error) {
                console.error('音频处理错误:', error);
                playNextChunk();
            }
        };
        
        playNextChunk();
    }
    
    // 打断当前音频
    interruptCurrentAudio() {
        if (this.currentAudio) {
            this.currentAudio.pause();
            this.currentAudio = null;
        }
        this.isAutoPlaying = false;
        this.currentMessageId = null;
    }
    
    // 停止所有音频播放
    stopAllAudio() {
        this.interruptCurrentAudio();
        this.audioMessages = {};
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
            connectionStatus: this.connectionStatus,
            isConnected: this.isConnected(),
            isRecording: this.isRecording,
            vadListening: this.vadListening,
            vadEnabled: this.vadEnabled,
            audioAutoPlayEnabled: this.audioAutoPlayEnabled,
            isAutoPlaying: this.isAutoPlaying,
            audioPlaybackUnlocked: this.audioPlaybackUnlocked
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
    }
}

// 导出
window.WebSocketManager = WebSocketManager; 