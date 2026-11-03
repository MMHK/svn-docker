<?php
/**
 * Bootstrap file for iF.SVNAdmin
 * Initializes the application environment and global services
 */

// Error reporting (PHP 8 compatibility patch - DO NOT REMOVE)
error_reporting(E_ALL & ~E_DEPRECATED);

// Load composer autoload
require_once __DIR__ . '/../vendor/autoload.php';

// Set exception handler
set_exception_handler('exception_handler');

// Check if config.ini exists
$configFile = './data/config.ini';
if (!file_exists($configFile)) {
    if (!copy('./data/config.tpl.ini', $configFile)) {
        throw new Exception("Could not copy configuration file template. Require write permission (777) to \"data\" folder.");
    }
    header("Location: settings.php");
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
$appCurrentScriptFile = currentScriptFileName();
if ($appEngine->isAuthenticationActive() && $appCurrentScriptFile != "settings.php") {
    if (!$appEngine->getAclManager()->hasAdminDefined()) {
        header("Location: settings.php");
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
