<?php
/**
 * 過渡期相容層：舊 FQCN → 新類別。
 *
 * 階段 2a 把 `classes/` 搬到 `src/` 並改 namespace，階段 2b-1 再把
 * `include/ifcorelib/IF_*` 搬進 `src/`（`Support\*`、`Config\*`、`View\*`、`I18n\*`、`Acl\*`）。
 * 這裡為每個被搬動的舊名稱保留入口，讓尚未改寫的舊程式碼（actions、pages、templates）
 * 在階段 3 結束前仍可運作。
 *
 * 使用者：`composer.json` 的 autoload.files，隨 vendor/autoload.php 一起載入。
 * 刪除時機：階段 3 結束（所有舊 FQCN 引用歸零）後整檔刪除。
 *
 * 注意：PHP 的類別名稱不分大小寫，`svnadmin\core\Engine` 與 `SvnAdmin\Core\Engine`
 * 其實是「同一個」類別，無法用 class_alias 建立別名（會報 name already in use）。
 * 這類「只差大小寫」的舊名稱見檔案末段的 compat autoloader。
 */

// 舊 FQCN 與新 FQCN 的大小寫以外確實不同者，才需要 class_alias。
$legacyAliases = array(
	// svnadmin\core\acl
	'svnadmin\core\acl\FSAclManager' => \SvnAdmin\Acl\FSAclManager::class,

	// svnadmin\core\entities
	'svnadmin\core\entities\AccessPath' => \SvnAdmin\Entities\AccessPath::class,
	'svnadmin\core\entities\Group' => \SvnAdmin\Entities\Group::class,
	'svnadmin\core\entities\Permission' => \SvnAdmin\Entities\Permission::class,
	'svnadmin\core\entities\Repository' => \SvnAdmin\Entities\Repository::class,
	'svnadmin\core\entities\RepositoryParent' => \SvnAdmin\Entities\RepositoryParent::class,
	'svnadmin\core\entities\RepositoryPath' => \SvnAdmin\Entities\RepositoryPath::class,
	'svnadmin\core\entities\Role' => \SvnAdmin\Entities\Role::class,
	'svnadmin\core\entities\User' => \SvnAdmin\Entities\User::class,

	// svnadmin\core\interfaces → SvnAdmin\Contracts
	'svnadmin\core\interfaces\IAclManager' => \SvnAdmin\Contracts\IAclManager::class,
	'svnadmin\core\interfaces\IAuthenticator' => \SvnAdmin\Contracts\IAuthenticator::class,
	'svnadmin\core\interfaces\IEditProvider' => \SvnAdmin\Contracts\IEditProvider::class,
	'svnadmin\core\interfaces\IGroupEditProvider' => \SvnAdmin\Contracts\IGroupEditProvider::class,
	'svnadmin\core\interfaces\IGroupViewProvider' => \SvnAdmin\Contracts\IGroupViewProvider::class,
	'svnadmin\core\interfaces\IPathsEditProvider' => \SvnAdmin\Contracts\IPathsEditProvider::class,
	'svnadmin\core\interfaces\IPathsViewProvider' => \SvnAdmin\Contracts\IPathsViewProvider::class,
	'svnadmin\core\interfaces\IProvider' => \SvnAdmin\Contracts\IProvider::class,
	'svnadmin\core\interfaces\IRepositoryEditProvider' => \SvnAdmin\Contracts\IRepositoryEditProvider::class,
	'svnadmin\core\interfaces\IRepositoryViewProvider' => \SvnAdmin\Contracts\IRepositoryViewProvider::class,
	'svnadmin\core\interfaces\IUserEditProvider' => \SvnAdmin\Contracts\IUserEditProvider::class,
	'svnadmin\core\interfaces\IUserViewProvider' => \SvnAdmin\Contracts\IUserViewProvider::class,
	'svnadmin\core\interfaces\IViewProvider' => \SvnAdmin\Contracts\IViewProvider::class,

	// 這兩個原本是全局類別（classes/core/Exceptions.class.php 沒有 namespace）。
	'CoreException' => \SvnAdmin\Core\CoreException::class,
	'ValidationException' => \SvnAdmin\Core\ValidationException::class,

	// 階段 2b-1：include/ifcorelib/IF_* → SvnAdmin\*
	'IF_Config' => \SvnAdmin\Config\Config::class,
	'IF_SyntaxErrorException' => \SvnAdmin\Config\SyntaxErrorException::class,
	'IF_IniFile' => \SvnAdmin\Config\IniFile::class,
	'IF_Locale' => \SvnAdmin\I18n\Locale::class,
	'IF_Translator' => \SvnAdmin\I18n\Translator::class,
	'IF_Template' => \SvnAdmin\View\Template::class,
	'IF_Template_Exception' => \SvnAdmin\View\TemplateException::class,
	'IF_HtPasswd' => \SvnAdmin\Support\HtPasswd::class,
	'IF_File' => \SvnAdmin\Support\File::class,
	'IF_StringUtils' => \SvnAdmin\Support\StringUtils::class,
	'IF_SVNBaseC' => \SvnAdmin\Support\SvnBase::class,
	'IF_SVNClientC' => \SvnAdmin\Support\SvnClient::class,
	'IF_SVNAdminC' => \SvnAdmin\Support\SvnAdmin::class,
	'IF_SVNAuthFileC' => \SvnAdmin\Support\SvnAuthFile::class,
	'IF_SVNException' => \SvnAdmin\Support\SvnException::class,
	'IF_SVNCommandExecutionException' => \SvnAdmin\Support\SvnCommandExecutionException::class,
	'IF_SVNOutputParseException' => \SvnAdmin\Support\SvnOutputParseException::class,
	'IF_SVNList' => \SvnAdmin\Support\SvnList::class,
	'IF_SVNListEntry' => \SvnAdmin\Support\SvnListEntry::class,
	'IF_ACL' => \SvnAdmin\Acl\Acl::class,
	'IF_ACLModule' => \SvnAdmin\Acl\AclModule::class,
	'IF_ACLRole' => \SvnAdmin\Acl\AclRole::class,
	'IF_ACL_Exception' => \SvnAdmin\Acl\AclException::class,
);

foreach ($legacyAliases as $legacyName => $newName) {
	if (class_exists($legacyName, false) || interface_exists($legacyName, false)) {
		continue;
	}
	class_alias($newName, $legacyName);
}
unset($legacyAliases, $legacyName, $newName);

/**
 * 只差大小寫的舊名稱：
 *   svnadmin\core\Engine              ↔ SvnAdmin\Core\Engine
 *   svnadmin\providers\*              ↔ SvnAdmin\Providers\*
 * PHP 類名不分大小寫，只要新類別已被載入，舊寫法自然解析到同一個類別。因此這裡只需
 * 補一個 autoloader：舊寫法先被用到時，替它把新類別載進來。
 */
$legacyCaseMap = array();
foreach (array(
	'svnadmin\core\Engine' => \SvnAdmin\Core\Engine::class,
	'svnadmin\providers\AuthFileGroupAndPathProvider' => \SvnAdmin\Providers\AuthFileGroupAndPathProvider::class,
	'svnadmin\providers\EngineBaseAuthenticator' => \SvnAdmin\Providers\EngineBaseAuthenticator::class,
	'svnadmin\providers\PasswdUserProvider' => \SvnAdmin\Providers\PasswdUserProvider::class,
	'svnadmin\providers\RepositoryEditProvider' => \SvnAdmin\Providers\RepositoryEditProvider::class,
	'svnadmin\providers\RepositoryViewProvider' => \SvnAdmin\Providers\RepositoryViewProvider::class,
) as $legacyName => $newName) {
	$legacyCaseMap[strtolower($legacyName)] = $newName;
}
unset($legacyName, $newName);

spl_autoload_register(static function ($class) use ($legacyCaseMap) {
	static $busy = false;
	if ($busy) {
		return;
	}
	$key = strtolower($class);
	if (!isset($legacyCaseMap[$key])) {
		return;
	}
	$busy = true;
	class_exists($legacyCaseMap[$key]);
	$busy = false;
});
unset($legacyCaseMap);

