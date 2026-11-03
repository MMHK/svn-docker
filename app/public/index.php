<?php
/**
 * Front controller for iF.SVNAdmin
 * All requests are routed through this file
 */

// Bootstrap the application
require_once __DIR__ . '/../src/bootstrap.php';

// Create HTTP objects
$request = new \SvnAdmin\Http\Request();
$router = new \SvnAdmin\Http\Router();

// TODO: Define routes in Phase 3b
// For now, just handle legacy .php files

// Create kernel and handle request
$kernel = new \SvnAdmin\Http\Kernel($router);
$response = $kernel->handle($request);
$kernel->terminate($response);
