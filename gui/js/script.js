$( document ).ready(function() {
    getId();
    // Update data with 1 second ticks
    setInterval(function() {
        getMessages();
    }, 1000);
    setInterval(function() {
        getNodes();
    }, 1000);

    $('#sendMessage').click(postMessage);
    $('#addNode').click(postNode);

    function getMessages() {
        let messagesElement = $('#messages');
        $.ajax({
            type: 'GET',
            url: '/message',
            data: '',
            success: function(response)
            {
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
        const GossipPacket = {
            Simple: {
                OriginalName: '',
                RelayPeerAddr: '',
                Contents: newMessage,
            }
        };
        const packet = JSON.stringify(GossipPacket);
        console.log(packet);
        $.ajax({
            type: 'POST',
            url: '/message',
            data: packet,
        });
        $('#newMessage').val('');
    }

    function getNodes() {
        let nodesElement = $('#nodes');
        $.ajax({
            type: 'GET',
            url: '/node',
            data: '',
            success: function(response)
            {
                if (response) {
                    const oldValue = nodesElement.text();
                    nodesElement.text('');
                    $.each(response, function(index, value) {
                        nodesElement.append(sanitizeString(value) + '\n');
                    });
                    if (oldValue !== nodesElement.text()) {
                        nodesElement.scrollTop(nodesElement.prop('scrollHeight'));
                    }
                }
            }
        });
    }

    function postNode() {
        const newNode = $('#newNode').val();
        if (isValidIPWithPort(newNode)) {
            const GossipPacket = {
                Node: newNode
            };
            const packet = JSON.stringify(GossipPacket);
            $.ajax({
                type: 'POST',
                url: '/node',
                data: newNode,
            });
        }
        $('#newNode').val('');
    }

    function getId() {
        let idElement = $('#peerId');
        $.ajax({
            type: 'GET',
            url: '/id',
            data: '',
            success: function(response)
            {
                if (response) {
                    idElement.val(response);
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
});