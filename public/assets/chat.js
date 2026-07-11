const MAX_CONNECT_RETRIES = 3;
const MAX_RECONNECT_ATTEMPTS = 3;

const UUID_NIL = "00000000-0000-0000-0000-000000000000";

function initChat() {
	const el = document.getElementById("chat_inner");
	if (!el) return;

	const peerID = el.dataset.peerId;
	const strangerPeerID = el.dataset.strangerPeerId;
	const isCaller = el.dataset.isCaller === "true";
	const video = el.dataset.video === "true";

	if (window.peer) {
		console.info("Destroying previous Peer object.");
		window.peer.destroy();
	}

	if (window._activeCall) {
		console.info("Closing previous call.");
		window._activeCall.close();
		window._activeCall = null;
	}

	if (!peerID || peerID === UUID_NIL) return;

	window._reconnectAttempts = 0;
	window._peerConnectRetries = 0;

	const currentPeer = new Peer(peerID, {
		host: "localhost",
		port: 9000,
		debug: 1,
		secure: false,
		config: {
			'iceServers': [
				{ urls: "stun:stun.l.google.com:19302" },
				{ urls: "turn:turn.bistri.com:80", username: "homeo", credential: "homeo" }
			]
		}
	});

	window.peer = currentPeer;

	currentPeer.on('error', err => {
		if (window.peer !== currentPeer) return;

		if (err.type === 'peer-unavailable' && isCaller) {
			console.error('Peer unable to connect with match, retrying to connect...');
			window._peerConnectRetries++;
			if (window._peerConnectRetries <= MAX_CONNECT_RETRIES) {
				setTimeout(() => {
					if (window.peer === currentPeer) {
						var newConn = currentPeer.connect(strangerPeerID);
						newConn.on('error', err => console.error('Connection error (retry):', err));
						newConn.on('open', () => {
							wsConnectedToPeerMessage();
							setupChat(newConn);
						});
					}
				}, 1000 * window._peerConnectRetries);
			} else {
				showError("Could not connect to stranger. Click New Chat to try again.");
			}
		}
	});

	currentPeer.on('disconnected', () => {
		if (window.peer !== currentPeer) return;

		console.error('Peer disconnected, attempting reconnect...');
		window._reconnectAttempts++;
		if (window._reconnectAttempts <= MAX_RECONNECT_ATTEMPTS) {
			currentPeer.reconnect();
		} else {
			showError("Could not re-connect to stranger. Click New Chat to try again.");
		}
	});

	currentPeer.on('open', id => {
		console.info('My peer ID is now registered with the server:', id);

		if (!strangerPeerID && strangerPeerID === UUID_NIL) return;

		if (isCaller) {
			// console.debug("Acting as CALLER. Connecting to:", { stranger: strangerPeerID });
			const conn = currentPeer.connect(strangerPeerID);

			conn.on('error', err => console.error('Connection error (caller):', { err }));
			conn.on('open', () => {
				wsConnectedToPeerMessage();
				setupChat(conn);
			});

		} else {
			// console.debug("Acting as RECEIVER. Waiting for connection...");
			currentPeer.on('connection', (conn) => {
				console.debug("Incoming connection received.");

				conn.on('error', err => console.error('Connection error (receiver):', { err }));
				conn.on('open', () => {
					wsConnectedToPeerMessage();
					setupChat(conn);
				});
			});
		}

		if (video) {
			const localVideo = document.querySelector("#local_video");
			const remoteVideo = document.querySelector("#remote_video");

			// Use previous stream if available.
			const streamPromise = window.localStream
				? Promise.resolve(window.localStream)
				: navigator.mediaDevices.getUserMedia({
					video: {
						width: { ideal: 640 },
						height: { ideal: 480 },
						frameRate: { ideal: 30 }
					},
					audio: {
						echoCancellation: true,
						noiseSuppression: true,
						autoGainControl: true
					}
				})
					.then(stream => {
						window.localStream = stream;
						return stream;
					}).catch(err => {
						console.error("Failed to get local stream:", err);
						showError("Camera or microphone access denied. Please allow permissions and try again.");
					});

			streamPromise.then(stream => {
				if (!stream) return;
				addVideoStream(localVideo, stream);
				localVideo.style.transform = 'scaleX(-1)';
				syncMediaButtons();
			});

			if (isCaller) {
				console.debug("📹 VIDEO CALL: initiated");
				streamPromise.then(stream => {
					if (!stream) return;
					const call = currentPeer.call(strangerPeerID, stream);
					window._activeCall = call;

					call.on('stream', partnerStream => {
						addVideoStream(remoteVideo, partnerStream);
						syncMediaButtonsRemote();
					});

					call.on('close', () => {
						if (remoteVideo) remoteVideo.srcObject = null;
						window._activeCall = null;
					});

					call.on('error', err => {
						console.error("📹 VIDEO CALL: (caller) ", { err });
					});
				});
			} else {
				console.debug("📹 VIDEO CALL: waiting for call");
				currentPeer.on('call', call => {
					console.log("📹 VIDEO CALL: received");
					window._activeCall = call;
					streamPromise.then(stream => {
						if (!stream) return;
						call.answer(stream);

						call.on('stream', callerStream => {
							addVideoStream(remoteVideo, callerStream);
							syncMediaButtonsRemote();
						});

						call.on('close', () => {
							if (remoteVideo) remoteVideo.srcObject = null;
							window._activeCall = null;
						});

						call.on('error', err => {
							console.error("📹 VIDEO CALL: (receiver) ", { err });
						});
					});
				});
			}
		}
	});
}

function setupChat(connection) {
	console.info('PeerJS DataConnection is open. Setting up chat handlers.');

	connection.on('data', data => {
		newChatBubble(data, false);
	});

	const chatForm = document.querySelector("#chat_form");
	const textarea = chatForm.querySelector("#chat_message");
	const sendChatButton = document.querySelector("#send_chat_button");

	const newButton = sendChatButton.cloneNode(true);
	sendChatButton.parentNode.replaceChild(newButton, sendChatButton);

	newButton.addEventListener("click", () => {
		const message = textarea.value.trim();
		if (message) {
			newChatBubble(message, true);
			connection.send(message);
			textarea.value = "";
		}
	});

	textarea.addEventListener('keydown', e => {
		if (e.key == 'Enter' && !e.shiftKey) {
			e.preventDefault();
			newButton.click();
		}
	});
}

function newChatBubble(message, isMe) {
	const wrapper = document.createElement("div");
	wrapper.classList.add("flex", "gap-1", "w-full", "text-sm", "sm:text-base", isMe ? "justify-end" : "justify-start");

	const innerWrapper = document.createElement("div");
	if (isMe) {
		innerWrapper.classList.add("flex", "gap-2", "p-2", "rounded-lg", "max-w-xs", "bg-blue-500", "text-white", "min-w-0");
	} else {
		innerWrapper.classList.add("flex", "gap-2", "p-2", "rounded-lg", "max-w-xs", "bg-gray-200", "text-gray-800", "min-w-0");
	}

	const msg = document.createElement("p");
	msg.classList.add("break-all", "whitespace-pre-line");
	msg.innerText = message;

	innerWrapper.append(msg);
	wrapper.appendChild(innerWrapper);

	const chatBubbles = document.querySelector("#chat_bubbles");
	chatBubbles.appendChild(wrapper);
	wrapper.scrollIntoView({ block: "end", behavior: "smooth" });
}

function wsConnectedToPeerMessage() {
	if (!window.socketWrapper) {
		console.error("Socket wrapper not found on window object.");
		return;
	}

	window.socketWrapper.send(JSON.stringify({ message_type: "peer_connected" }));
}

function showError(msg) {
	const el = document.getElementById("error_message");
	if (el) {
		document.getElementById("error_text").innerText = msg;
		el.classList.remove("hidden");
	}
}

function addVideoStream(videoElement, stream) {
	videoElement.srcObject = stream
	videoElement.addEventListener('loadedmetadata', () => {
		videoElement.play();
	});
}

function syncMediaButtons() {
	const audioTrack = window.localStream?.getAudioTracks()[0];
	const muteBtn = document.getElementById("mute_button");
	if (muteBtn && audioTrack) {
		muteBtn.innerHTML = `<svg class="w-5 h-5 fill-white"><use href="/assets/icons.svg#${audioTrack.enabled ? "mic-on" : "mic-off"}"></use></svg>`;
	}

	const videoTrack = window.localStream?.getVideoTracks()[0];
	const camBtn = document.getElementById("camera_button");
	if (camBtn && videoTrack) {
		camBtn.innerHTML = `<svg class="w-5 h-5 fill-white"><use href="/assets/icons.svg#${videoTrack.enabled ? "cam-on" : "cam-off"}"></use></svg>`;
	}
}

function toggleMute() {
	const track = window.localStream?.getAudioTracks()[0];
	if (!track) return;
	track.enabled = !track.enabled;
	syncMediaButtons();
}

function toggleCamera() {
	const track = window.localStream?.getVideoTracks()[0];
	if (!track) return;
	track.enabled = !track.enabled;

	// to remove the last freezed frame from the video element
	const localVideo = document.querySelector("#local_video");
	if (localVideo) {
		localVideo.srcObject = track.enabled ? window.localStream : null;
	}

	syncMediaButtons();
}

function syncMediaButtonsRemote() {
	let isMuted = window._remoteMuted;
	if (typeof isMuted == "undefined") {
		isMuted = true;
		window._remoteMuted = true;
	}

	const remoteVideo = document.querySelector("#remote_video");
	if (!remoteVideo) return;

	remoteVideo.muted = isMuted;

	const remoteMuteBtn = document.querySelector("#remote_mute_button");
	if (!remoteMuteBtn) return;

	remoteMuteBtn.innerHTML = `<svg class="w-5 h-5 fill-white"><use href="/assets/icons.svg#${isMuted ? "muted" : "unmuted"}"></use></svg>`;
}

function toggleMuteRemote() {
	const remoteVideo = document.querySelector("#remote_video");
	if (!remoteVideo) return;

	remoteVideo.muted = !remoteVideo.muted;
	window._remoteMuted = remoteVideo.muted;
	syncMediaButtonsRemote()
}
