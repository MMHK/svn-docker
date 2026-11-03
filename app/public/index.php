<?php
/**
 * Front controller for iF.SVNAdmin
 * All requests are routed through this file
 */

// Bootstrap the application
require_once __DIR__ . '/../src/bootstrap.php';

// Create HTTP objects
$request = new \SvnAdmin\Http\Request();
$response = new \SvnAdmin\Http\Response();
$router = new \SvnAdmin\Http\Router();

// Helper function to create controller instance
$createController = function($controllerClass) use ($request, $response) {
    return new $controllerClass($request, $response);
};

// ============================================================================
// Home routes
// ============================================================================
$router->get('/', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\HomeController::class)->handle();
});
$router->get('/index.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\HomeController::class)->handle();
});

// ============================================================================
// Login routes
// ============================================================================
$router->any('/login', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\LoginController::class)->handle();
});
$router->any('/login.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\LoginController::class)->handle();
});

// ============================================================================
// Logout routes
// ============================================================================
$router->any('/logout', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\LogoutController::class)->handle();
});
$router->any('/logout.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\LogoutController::class)->handle();
});

// ============================================================================
// User routes
// ============================================================================
$router->any('/users', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\UserController::class)->list();
});
$router->any('/userlist.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\UserController::class)->list();
});

$router->any('/users/new', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\UserController::class)->create();
});
$router->any('/usercreate.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\UserController::class)->create();
});

$router->any('/users/view', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\UserController::class)->view();
});
$router->any('/userview.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\UserController::class)->view();
});

$router->any('/users/change-password', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\UserController::class)->changePassword();
});
$router->any('/userchangepass.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\UserController::class)->changePassword();
});

$router->any('/users/group-assign', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\MembershipController::class)->handle();
});
$router->any('/usergroupassign.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\MembershipController::class)->handle();
});

// ============================================================================
// Group routes
// ============================================================================
$router->any('/groups', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\GroupController::class)->list();
});
$router->any('/grouplist.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\GroupController::class)->list();
});

$router->any('/groups/new', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\GroupController::class)->create();
});
$router->any('/groupcreate.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\GroupController::class)->create();
});

$router->any('/groups/view', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\GroupController::class)->view();
});
$router->any('/groupview.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\GroupController::class)->view();
});

// ============================================================================
// Repository routes
// ============================================================================
$router->any('/repositories', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\RepositoryController::class)->list();
});
$router->any('/repositorylist.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\RepositoryController::class)->list();
});

$router->any('/repositories/new', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\RepositoryController::class)->create();
});
$router->any('/repositorycreate.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\RepositoryController::class)->create();
});

$router->any('/repositories/view', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\RepositoryController::class)->view();
});
$router->any('/repositoryview.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\RepositoryController::class)->view();
});

// ============================================================================
// Access Path routes
// ============================================================================
$router->any('/access-paths', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\AccessPathController::class)->list();
});
$router->any('/accesspathslist.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\AccessPathController::class)->list();
});

$router->any('/access-paths/new', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\AccessPathController::class)->create();
});
$router->any('/accesspathcreate.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\AccessPathController::class)->create();
});

$router->any('/access-paths/view', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\AccessPathController::class)->view();
});
$router->any('/accesspathview.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\AccessPathController::class)->view();
});

// ============================================================================
// Permission routes
// ============================================================================
$router->any('/permissions', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\PermissionController::class)->handle();
});
$router->any('/permissionassign.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\PermissionController::class)->handle();
});

// ============================================================================
// Role routes
// ============================================================================
$router->any('/roles', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\RoleController::class)->handle();
});
$router->any('/rolelist.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\RoleController::class)->handle();
});

// ============================================================================
// Settings routes
// ============================================================================
$router->any('/settings', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\SettingsController::class)->handle();
});
$router->any('/settings.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\SettingsController::class)->handle();
});

// ============================================================================
// Error routes
// ============================================================================
$router->any('/error', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\ErrorController::class)->handle();
});
$router->any('/error.php', function() use ($createController) {
    return $createController(\SvnAdmin\Controllers\ErrorController::class)->handle();
});

// Create kernel and handle request
$kernel = new \SvnAdmin\Http\Kernel($router);
$response = $kernel->handle($request);
$kernel->terminate($response);
