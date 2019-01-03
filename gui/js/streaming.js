let player,
    url = '/streaming/',
    assetURL = '',
    mimeType = 'audio/mpeg',
    totalSegments = 10,
    segmentLength = 0,
    segmentDuration = 0,
    bytesFetched = 0,
    realSize = 0,
    requestedSegments = [],
    sourceBuffer = null,
    mediaSource = null;

function getInfoForStreaming() {
    const fileName = $('#streamFileName').val();
    assetURL = url + fileName;

    $.ajax
    ({
        type: "POST",
        url: '/streamFileInfo',
        data: fileName,
        success: function (data) {
            const parsedData = JSON.parse(data);
            console.log("Information:", parsedData);
            if (data && parsedData && parsedData.Name && parsedData.Size) {
                startStreaming(parsedData.Name, parsedData.Size);
            } else {
                alert("Error, streamable file not valid.");
            }
        },
        error: function () {
            alert("Error, streamable file not found.");
        }
    });
}

function startStreaming(fileName, size) {
    player = document.getElementById("audioPlayer");
    segmentLength = 0;
    segmentDuration = 0;
    bytesFetched = 0;
    realSize = 0;
    requestedSegments = [];
    sourceBuffer = null;
    mediaSource = null;

    realSize = size;
    for (let i = 0; i < totalSegments; i++) {
        requestedSegments[i] = false;
    }
    if ('MediaSource' in window && MediaSource.isTypeSupported(mimeType)) {
        // mediaSource = new MediaSource;
        // player.src = URL.createObjectURL(mediaSource);
        // mediaSource.addEventListener('sourceopen', onSourceOpen);

        player.src = assetURL;
    } else {
        console.error('Unsupported MIME type or codec: ', mimeType);
    }
}

function onSourceOpen() {
    sourceBuffer = mediaSource.addSourceBuffer(mimeType);
    getFileLength(assetURL, function (fileLength) {
        console.log("sizes: ", fileLength, realSize);
        console.log((realSize / 1024 / 1024).toFixed(2), 'MB');
        segmentLength = Math.round(realSize / totalSegments);
        fetchRange(assetURL, 0, segmentLength);
        requestedSegments[0] = true;
        player.addEventListener('timeupdate', checkBuffer);
        player.addEventListener('canplay', function () {
            console.log("canplayyyy:", player.duration, totalSegments);
            segmentDuration = player.duration / totalSegments;
            player.play();
        });
        player.ondurationchange = function () {
            console.log('durationchange:', player.duration);
        };
        player.onloadedmetadata = function () {
            console.log('onlodademetadata:', player.duration);
        };
        player.addEventListener('seeking', seek);
    });
}


function getFileLength(url, cb) {
    var xhr = new XMLHttpRequest;
    xhr.open('head', assetURL);
    xhr.onload = function () {
        cb(xhr.getResponseHeader('content-length'));
    };
    xhr.send();
}

function fetchRange(url, start, end) {
    let xhr = new XMLHttpRequest;
    xhr.open('get', url);
    xhr.responseType = 'arraybuffer';
    // xhr.setRequestHeader('Range', 'bytes=' + start + '-' + end);
    xhr.onload = function () {
        console.log('Fetched range: ', start, end);
        bytesFetched += end - start + 1;
        sourceBuffer.appendBuffer(xhr.response);
    };
    xhr.send();
}

function checkBuffer() {
    let currentSegment = getCurrentSegment();
    console.log(currentSegment);
    console.log(player.currentTime);
    if (currentSegment === totalSegments && isAllLoaded()) {
        console.log('All segments loaded', mediaSource.readyState);
        mediaSource.endOfStream();
        player.removeEventListener('timeupdate', checkBuffer);
    } else if (isNextSegmentFetchNeeded(currentSegment)) {
        console.log("next neddee");
        requestedSegments[currentSegment] = true;
        console.log('Player time to fetch next chunk', player.currentTime);
        fetchRange(assetURL, bytesFetched, bytesFetched + segmentLength);
    }
}

function seek(event) {
    console.log("seeek");
    console.log(event);
    if (mediaSource.readyState === 'open') {
        sourceBuffer.abort();
        console.log(mediaSource.readyState);
    } else {
        console.log('Error: Seeking but MediaSource is not open');
        console.log(mediaSource.readyState);
    }
}

function getCurrentSegment() {
    return ((player.currentTime / segmentDuration) | 0) + 1;
}

function isAllLoaded() {
    return requestedSegments.every(function (val) {
        return !!val
    });
}

function isNextSegmentFetchNeeded(currentSegment) {
    console.log("current seg: ", currentSegment, " currentTIme: ", player.currentTime, " threshold: ", segmentDuration * currentSegment * 0.7, " isrequ: ", requestedSegments[currentSegment]);
    return player.currentTime > segmentDuration * currentSegment * 0.7 && !requestedSegments[currentSegment];
}
