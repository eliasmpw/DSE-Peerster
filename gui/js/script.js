$(document).ready(function () {
    getId();
    getIpAddress();
    getMessages();
    getPeerNodes();
    getAllNodes();
    // Update data with 1 or 2 seconds ticks
    setInterval(function () {
        getMessages();
    }, 1000);
    setInterval(function () {
        getPeerNodes();
    }, 2000);
    setInterval(function () {
        getAllNodes();
    }, 2000);

    $('#sendMessage').click(postMessage);
    $('#addPeerNode').click(postPeerNode);
    $('#sendPrivateMessage').click(postPrivateMessage);
    $('#newMessage').keyup(enableSendMessageBtn);
    $('#newPeerNode').keyup(enableNewPeerNodeBtn);
    $('#newPrivateMessage').keyup(enableSendPrivateMessageBtn);
    $('#privateMessageModal').on('show.bs.modal', onModalOpened);
    $('#privateMessageModal').on('hide.bs.modal', onModalClosed);
    $('#shareFile').click(shareFileBtn);
    $('#downloadFileButton').click(downloadFileBtn);
    $('#searchFileButton').click(searchFileBtn);
    $('body').on('click', '.searchResultFile', downloadFileFromSearch);

    let idName;
    let ipAddress;
    let selectedPrivateName;
    let selectedPrivateAddress;
    let privateMessageInterval;

    function getMessages() {
        let messagesElement = $('#messages');
        $.ajax({
            type: 'GET',
            url: '/message',
            data: '',
            success: function (response) {
                if (response) {
                    const oldValue = messagesElement.text();
                    messagesElement.text('');
                    $.each(response, function (key, value) {
                        messagesElement.append(sanitizeString(value.Origin) + ': ' +
                            sanitizeString(value.Text) + '\n');
                    });
                    if (oldValue !== messagesElement.text()) {
                        messagesElement.scrollTop(messagesElement.prop('scrollHeight'));
                    }
                }
            }
        });
    }

    function postMessage() {
        const newMessage = $('#newMessage').val();
        if (newMessage !== '') {
            const GossipPacket = {
                Simple: {
                    OriginalName: '',
                    RelayPeerAddr: '',
                    Contents: newMessage,
                }
            };
            const packet = JSON.stringify(GossipPacket);
            console.log("POST /message: ", packet);
            $.ajax({
                type: 'POST',
                url: '/message',
                data: packet,
            });
        }
        $('#newMessage').val('');
        enableSendMessageBtn();
    }

    function getPeerNodes() {
        let nodesElement = $('#peerNodes');
        $.ajax({
            type: 'GET',
            url: '/node',
            data: '',
            success: function (response) {
                if (response) {
                    const oldValue = nodesElement.text();
                    nodesElement.text('');
                    $.each(response, function (index, value) {
                        nodesElement.append(sanitizeString(value) + '\n');
                    });
                    if (oldValue !== nodesElement.text()) {
                        nodesElement.scrollTop(nodesElement.prop('scrollHeight'));
                    }
                }
            }
        });
    }

    function postPeerNode() {
        const newPeerNode = $('#newPeerNode').val();
        if (isValidIPWithPort(newPeerNode)) {
            const GossipPacket = {
                Node: newPeerNode
            };
            const packet = JSON.stringify(GossipPacket);
            $.ajax({
                type: 'POST',
                url: '/node',
                data: newPeerNode,
            });
        }
        $('#newPeerNode').val('');
        enableNewPeerNodeBtn();
    }

    function getId() {
        let idElement = $('#peerId');
        $.ajax({
            type: 'GET',
            url: '/id',
            data: '',
            success: function (response) {
                if (response) {
                    idName = response;
                    idElement.val(response);
                }
            }
        });
    }

    function getIpAddress() {
        $.ajax({
            type: 'GET',
            url: '/ipAddress',
            data: '',
            success: function (response) {
                if (response) {
                    ipAddress = response;
                }
            }
        });
    }

    function isValidIPWithPort(value) {
        let parts = value.split(":");
        let ipAddress = parts[0].split(".");
        let port = parts[1];
        return isBetween(port, 1, 65535) && ipAddress.length === 4 && ipAddress.every(function (segment) {
            return isBetween(segment, 0, 255);
        });
    }

    function isBetween(value, min, max) {
        const num = +value;
        return num >= min && num <= max;
    }

    function sanitizeString(myString) {
        return myString
            .replace(/&/g, '&amp;')
            .replace(/>/g, '&gt;')
            .replace(/</g, '&lt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#039;');
    }

    function enableSendMessageBtn() {
        $('#sendMessage').prop('disabled', $('#newMessage').val() == '');
    }

    function enableNewPeerNodeBtn() {
        $('#addPeerNode').prop('disabled', !isValidIPWithPort($('#newPeerNode').val()));
    }

    function enableSendPrivateMessageBtn() {
        $('#sendPrivateMessage').prop('disabled', $('#newPrivateMessage').val() == '');
    }

    function getAllNodes() {
        let nodesElement = $('#allNodes');
        $.ajax({
            type: 'GET',
            url: '/allNodes',
            data: '',
            success: function (response) {
                if (response) {
                    newContent = "";
                    for (let knownNode in response) {
                        newContent = newContent + '<div class="knownNode" ' +
                            'data-toggle="modal" data-target="#privateMessageModal" ' +
                            'data-name="' + knownNode + '" ' +
                            'data-address="' + response[knownNode] + '">' + knownNode + '</div>';
                    }
                    nodesElement.html(newContent);
                }
            }
        });
    }

    function getPrivateMessages() {
        let messagesElement = $('#privateMessages');
        $.ajax({
            type: 'GET',
            url: '/privateMessage',
            data: '',
            success: function (response) {
                if (response) {
                    const oldValue = messagesElement.text();
                    response = response.filter(message => {
                        return message.Origin === idName && message.Destination === selectedPrivateName ||
                            message.Origin === selectedPrivateName && message.Destination === idName;
                    });
                    messagesElement.text('');
                    $.each(response, function (key, value) {
                        messagesElement.append(sanitizeString(value.Origin) + ': ' +
                            sanitizeString(value.Text) + '\n');
                    });
                    if (oldValue !== messagesElement.text()) {
                        messagesElement.scrollTop(messagesElement.prop('scrollHeight'));
                    }
                }
            }
        });
    }

    function postPrivateMessage() {
        const newMessage = $('#newPrivateMessage').val();
        if (newMessage !== '') {
            const GossipPacket = {
                Private: {
                    Origin: '',
                    ID: 0,
                    Text: newMessage,
                    Destination: selectedPrivateName,
                    HopLimit: 10
                }
            };
            const packet = JSON.stringify(GossipPacket);
            console.log("POST /privateMessage: ", packet);
            $.ajax({
                type: 'POST',
                url: '/privateMessage',
                data: packet,
            });
        }
        $('#newPrivateMessage').val('');
        enableSendPrivateMessageBtn();
    }

    function onModalOpened(event) {
        selectedPrivateName = event.relatedTarget.dataset.name;
        selectedPrivateAddress = event.relatedTarget.dataset.address;
        getPrivateMessages();
        privateMessageInterval = setInterval(function () {
            getPrivateMessages();
        }, 1000);
    }

    function onModalClosed(event) {
        clearInterval(privateMessageInterval);
    }

    function shareFileBtn() {
        const filePath = $('#selectedFile').val();
        if (filePath) {
            $.ajax({
                type: 'POST',
                url: '/shareFile',
                data: filePath,
            });
        }
        $('#selectedFile').val('');
    }

    function downloadFileBtn() {
        const fileName = $('#downloadFileName').val();
        const hash = $('#downloadHash').val();
        const node = $('#downloadNode').val();
        if (fileName && hash && node) {
            const GossipPacket = {
                DataRequest: {
                    Origin: '',
                    Destination: node,
                    HopLimit: 10,
                    HashValue: decodeHex(hash),
                    FileName: fileName
                }
            };
            console.log(GossipPacket);
            const packet = JSON.stringify(GossipPacket);
            $('#downloadInfo').show();
            $.ajax({
                type: 'POST',
                url: '/downloadFile',
                data: packet,
                success: function (data, text) {
                    $('#downloadInfo').hide();
                    alert("Download of " + fileName + " completed!");
                },
                error: function (request, status, error) {
                    $('#downloadInfo').hide();
                    alert("Error in download");
                }
            });
        }
        $('#downloadFileName').val('');
        $('#downloadHash').val('');
        $('#downloadNode').val('');
    }

    function searchFileBtn() {
        const keywords = parseKeywords($('#searchKeyword').val());
        if (keywords) {
            const GossipPacket = {
                SearchRequest: {
                    Origin: '',
                    Budget: 2,
                    Keywords: keywords
                }
            };
            console.log(GossipPacket);
            const packet = JSON.stringify(GossipPacket);
            $('#searchInfo').show();
            $.ajax({
                type: 'POST',
                url: '/searchFile',
                data: packet,
                success: function (response) {
                    console.log("response: ", response);
                    if (response) {
                        newContent = "";
                        for (let metafile of response) {
                            console.log(metafile);
                            newContent = newContent + '<div class="searchResultFile" data-hashvalue="' + metafile.HashValue + '" data-nodename="' + metafile.Origins[0] + '">' + metafile.Name + '</div>';
                        }
                        $('#allFilesSearch').html(newContent);
                    }
                    $('#searchInfo').hide();
                },
                error: function (request, status, error) {
                    $('#searchInfo').hide();
                }
            });
        }
        $('#searchKeyword').val('');
    }

    function downloadFileFromSearch(element) {
        const fileName = element.target.innerHTML;
        const hash = element.target.dataset.hashvalue;
        const node = element.target.dataset.nodename;
        if (fileName && hash && node) {
            const GossipPacket = {
                DataRequest: {
                    Origin: '',
                    Destination: node,
                    HopLimit: 10,
                    HashValue: hash,
                    FileName: fileName
                }
            };
            console.log(GossipPacket);
            const packet = JSON.stringify(GossipPacket);
            $('#downloadInfo').show();
            $.ajax({
                type: 'POST',
                url: '/downloadFile',
                data: packet,
                success: function (data, text) {
                    $('#downloadInfo').hide();
                    alert("Download of " + fileName + " completed!");
                },
                error: function (request, status, error) {
                    $('#downloadInfo').hide();
                    alert("Error in download");
                }
            });
        }
    }

    function decodeHex(myString) {
        let bytes = [];
        for (let i = 0; i < myString.length; i += 2)
            bytes.push(parseInt(myString.substr(i, 2), 16));
        return bytes;
    }

    function parseKeywords(keywords) {
        keywords = keywords.trim();
        if (keywords) {
            return keywords.split(',');
        }
        return null
    }
});