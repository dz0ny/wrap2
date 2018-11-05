<?php

class Logger
{
    protected static $instance = null;
    const path = "./log.sock";

    public function __construct()
    {
        $this->socket = socket_create(AF_UNIX, SOCK_STREAM, 1);
        socket_set_nonblock($this->socket);
        socket_connect($this->socket, self::path);
    }

    public function __destruct()
    {
        socket_close($this->socket);
    }

    public function send($item)
    {
        $payload = json_encode($item);
        socket_write($this->socket, $payload);
    }

    protected function __clone()
    {
        //Prevent new instances
    }

    public static function log($item)
    {
        if (!isset(static::$instance)) {
            static::$instance = new static;
        }
        return (static::$instance)->send($item);
    }
}

Logger::log([1, "foo"]);
