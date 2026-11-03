<?php
/**
 * Bootstrap file for iF.SVNAdmin
 * Initializes the application environment and global services
 */

// Error reporting (PHP 8 compatibility patch - DO NOT REMOVE)
error_reporting(E_ALL & ~E_DEPRECATED);

// Load composer autoload
require_once __DIR__ . '/../vendor/autoload.php';

// 前端控制器把 CWD 移到 public/（實際為 Apache 的 /var/www），這裡切回 app 根，
// 讓既有相對路徑（./data、./translations、./data/backup）維持原行為
chdir(dirname(__DIR__));

// Set exception handler
set_exception_handler('exception_handler');

// Check if config.ini exists
$configFile = './data/config.ini';
if (!file_exists($configFile)) {
    if (!copy('./data/config.tpl.ini', $configFile)) {
        throw new Exception("Could not copy configuration file template. Require write permission (777) to \"data\" folder.");
    }
    header("Location: " . app_url("settings.php"));
    exit(0);
}

// Handle locale switching from request
$requestedLocale = get_request_var("locale");
if ($requestedLocale != null) {
    $_COOKIE["locale"] = $requestedLocale;
    @setcookie("locale", $requestedLocale, time()+60*60*24*365);
}

// Initialize global services
global $appEngine, $appTR, $appTemplate;

$appEngine = \SvnAdmin\Core\Engine::getInstance();
$appTR = \SvnAdmin\I18n\Translator::getInstance();
$appTR->setTranslationDirectory($appEngine->getConfig()->getValue("Translation", "Directory"));
$appTemplate = new \SvnAdmin\View\Template();

// Initialize providers for this deployment
\SvnAdmin\Core\ProviderFactory::initialize($appEngine, $appTemplate);

// Check if administrator is defined (when auth is active)
// 前端控制器下 SCRIPT_NAME 永遠是 index.php，改用請求路徑判斷是否在設定頁，
// 否則首次啟動會不斷 Redirect 到 settings 造成迴圈
$requestPath = parse_url($_SERVER['REQUEST_URI'] ?? '/', PHP_URL_PATH) ?? '/';
$isSettingsPage = str_contains($requestPath, 'settings');
if ($appEngine->isAuthenticationActive() && !$isSettingsPage) {
    if (!$appEngine->getAclManager()->hasAdminDefined()) {
        // 相對於應用根的設定頁（不要寫死 settings.php 或絕對路徑）
        header("Location: " . app_url("settings"));
        exit(0);
    }
}

// Set user locale from cookie or default
if (isset($_COOKIE["locale"]) && !empty($_COOKIE["locale"])) {
    $appTR->setCurrentLocale($_COOKIE["locale"]);
} else {
    $appTR->setCurrentLocale($appEngine->getConfig()->getValue("GUI", "DefaultLocale", "en_US"));
}

// Set global template variables
SetValue("LocaleList", $appTR->getAvailableLocales());

// Load global translation module
$appTR->loadModule("global");
