<?php
require_once __DIR__ . '/../../config/config.php';
require_once SRC_PATH . '/Auth.php';

// Check authentication
session_start();
$auth = new Auth();
if (!$auth->isLoggedIn()) {
    header('HTTP/1.1 401 Unauthorized');
    exit('Unauthorized');
}

$currentUser = $auth->getCurrentUser();
if ($currentUser['role'] !== 'admin') {
    header('HTTP/1.1 403 Forbidden');
    exit('Admin access required');
}

// WebSocket handshake
$headers = [];
foreach ($_SERVER as $key => $value) {
    if (substr($key, 0, 5) === 'HTTP_') {
        $header = str_replace(' ', '-', ucwords(strtolower(str_replace('_', ' ', substr($key, 5)))));
        $headers[$header] = $value;
    }
}

if (!isset($headers['Upgrade']) || strtolower($headers['Upgrade']) !== 'websocket') {
    header('HTTP/1.1 400 Bad Request');
    exit('WebSocket upgrade required');
}

if (!isset($headers['Sec-Websocket-Key'])) {
    header('HTTP/1.1 400 Bad Request');
    exit('Missing Sec-WebSocket-Key');
}

// Perform WebSocket handshake
$key = $headers['Sec-Websocket-Key'];
$acceptKey = base64_encode(sha1($key . '258EAFA5-E914-47DA-95CA-C5AB0DC85B11', true));

header('HTTP/1.1 101 Switching Protocols');
header('Upgrade: websocket');
header('Connection: Upgrade');
header('Sec-WebSocket-Accept: ' . $acceptKey);

// Flush headers
ob_end_flush();

// Get the socket
$socket = fopen('php://input', 'r');
$output = fopen('php://output', 'w');

// Start shell process
$descriptorspec = [
    0 => ['pipe', 'r'],  // stdin
    1 => ['pipe', 'w'],  // stdout
    2 => ['pipe', 'w']   // stderr
];

$process = proc_open(
    'bash --login',
    $descriptorspec,
    $pipes,
    '/root'
);

if (!is_resource($process)) {
    exit('Failed to start shell');
}

stream_set_blocking($pipes[0], false);
stream_set_blocking($pipes[1], false);
stream_set_blocking($pipes[2], false);
stream_set_blocking($socket, false);

// Send welcome message
$welcome = "\x1b[32mServerOS Web Terminal\x1b[0m\r\n";
$welcome .= "Connected as: \x1b[33m" . $currentUser['username'] . "\x1b[0m\r\n";
$welcome .= "Type 'exit' to close the session.\r\n\r\n";
fwrite($pipes[0], "export PS1='\\[\\e[32m\\]\\u@\\h\\[\\e[0m\\]:\\[\\e[34m\\]\\w\\[\\e[0m\\]\\$ '\n");
sendFrame($output, $welcome);

$lastActivity = time();
$timeout = 3600; // 1 hour timeout

while (true) {
    // Check for client data
    $frame = readFrame($socket);
    if ($frame !== null) {
        $lastActivity = time();
        
        $data = json_decode($frame, true);
        if ($data && isset($data['type'])) {
            if ($data['type'] === 'input') {
                fwrite($pipes[0], $data['data']);
            } elseif ($data['type'] === 'resize') {
                // Handle terminal resize
                if (isset($data['cols']) && isset($data['rows'])) {
                    // Note: This requires stty, which should be available
                }
            }
        }
    }

    // Read from shell stdout
    $output_data = fread($pipes[1], 8192);
    if ($output_data !== false && strlen($output_data) > 0) {
        sendFrame($output, $output_data);
        $lastActivity = time();
    }

    // Read from shell stderr
    $error_data = fread($pipes[2], 8192);
    if ($error_data !== false && strlen($error_data) > 0) {
        sendFrame($output, "\x1b[31m" . $error_data . "\x1b[0m");
        $lastActivity = time();
    }

    // Check if shell process has ended
    $status = proc_get_status($process);
    if (!$status['running']) {
        sendFrame($output, "\r\n\x1b[31mSession ended.\x1b[0m\r\n");
        break;
    }

    // Check for timeout
    if (time() - $lastActivity > $timeout) {
        sendFrame($output, "\r\n\x1b[33mSession timeout.\x1b[0m\r\n");
        break;
    }

    // Small delay to prevent CPU spinning
    usleep(10000); // 10ms
}

// Cleanup
fclose($pipes[0]);
fclose($pipes[1]);
fclose($pipes[2]);
proc_terminate($process);
proc_close($process);

function readFrame($socket) {
    $header = fread($socket, 2);
    if (strlen($header) < 2) {
        return null;
    }

    $byte1 = ord($header[0]);
    $byte2 = ord($header[1]);

    $opcode = $byte1 & 0x0F;
    $masked = ($byte2 & 0x80) !== 0;
    $payloadLen = $byte2 & 0x7F;

    if ($opcode === 0x08) { // Close frame
        return false;
    }

    if ($payloadLen === 126) {
        $extended = fread($socket, 2);
        $payloadLen = unpack('n', $extended)[1];
    } elseif ($payloadLen === 127) {
        $extended = fread($socket, 8);
        $payloadLen = unpack('J', $extended)[1];
    }

    if ($masked) {
        $mask = fread($socket, 4);
    }

    $payload = '';
    $remaining = $payloadLen;
    while ($remaining > 0) {
        $chunk = fread($socket, $remaining);
        if ($chunk === false || strlen($chunk) === 0) {
            break;
        }
        $payload .= $chunk;
        $remaining -= strlen($chunk);
    }

    if ($masked) {
        $decoded = '';
        for ($i = 0; $i < strlen($payload); $i++) {
            $decoded .= $payload[$i] ^ $mask[$i % 4];
        }
        return $decoded;
    }

    return $payload;
}

function sendFrame($socket, $data) {
    $length = strlen($data);
    $header = chr(0x81); // Text frame

    if ($length <= 125) {
        $header .= chr($length);
    } elseif ($length <= 65535) {
        $header .= chr(126) . pack('n', $length);
    } else {
        $header .= chr(127) . pack('J', $length);
    }

    fwrite($socket, $header . $data);
    fflush($socket);
}

