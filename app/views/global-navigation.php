<?php
/**
 * 全域水平導航（GitHub repo tab 風格）。
 * 連結使用新路由，舊的 *.php 路由仍由前端控制器保留可用。
 */
$svnNavBasePath = rtrim(str_replace('\\', '/', dirname($_SERVER['SCRIPT_NAME'] ?? '')), '/');
$svnNavPath = parse_url($_SERVER['REQUEST_URI'] ?? '/', PHP_URL_PATH) ?? '/';
if ($svnNavBasePath !== '' && str_starts_with($svnNavPath, $svnNavBasePath)) {
	$svnNavPath = substr($svnNavPath, strlen($svnNavBasePath));
}
if ($svnNavPath === '') {
	$svnNavPath = '/';
}
$svnNavIs = function (array $prefixes) use ($svnNavPath): bool {
	foreach ($prefixes as $prefix) {
		if (str_starts_with($svnNavPath, $prefix)) {
			return true;
		}
	}
	return false;
};
?>
<nav id="global-navigation" class="gh-tabs" aria-label="<?php Translate("Home"); ?>">
	<a class="gh-tab" href="./"<?php if ($svnNavPath === '/' || str_starts_with($svnNavPath, '/index.php')) { ?> aria-current="page"<?php } ?>>
		<?php Translate("Home"); ?>
	</a>

	<?php if (IsProviderActive(PROVIDER_REPOSITORY_VIEW) && (HasAccess(ACL_MOD_REPO, ACL_ACTION_VIEW) || HasAccess(ACL_MOD_REPO, ACL_ACTION_ADD))) { ?>
	<a class="gh-tab" href="repositories"<?php if ($svnNavIs(array('/repositories', '/repositorylist.php', '/repositorycreate.php', '/repositoryview.php'))) { ?> aria-current="page"<?php } ?>>
		<?php Translate("Repositories"); ?>
	</a>
	<?php } ?>

	<?php if (IsProviderActive(PROVIDER_USER_VIEW) && (HasAccess(ACL_MOD_USER, ACL_ACTION_VIEW) || HasAccess(ACL_MOD_USER, ACL_ACTION_ADD))) { ?>
	<a class="gh-tab" href="users"<?php if ($svnNavIs(array('/users', '/userlist.php', '/usercreate.php', '/userview.php', '/userchangepass.php'))) { ?> aria-current="page"<?php } ?>>
		<?php Translate("Users"); ?>
	</a>
	<?php } ?>

	<?php if (IsProviderActive(PROVIDER_GROUP_VIEW) && (HasAccess(ACL_MOD_GROUP, ACL_ACTION_VIEW) || HasAccess(ACL_MOD_GROUP, ACL_ACTION_ADD) || HasAccess(ACL_MOD_GROUP, ACL_ACTION_ASSIGN))) { ?>
	<a class="gh-tab" href="groups"<?php if ($svnNavIs(array('/groups', '/grouplist.php', '/groupcreate.php', '/groupview.php', '/usergroupassign.php'))) { ?> aria-current="page"<?php } ?>>
		<?php Translate("Groups"); ?>
	</a>
	<?php } ?>

	<?php if (IsProviderActive(PROVIDER_ACCESSPATH_VIEW) && (HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_VIEW) || HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_ADD) || HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_ASSIGN))) { ?>
	<a class="gh-tab" href="access-paths"<?php if ($svnNavIs(array('/access-paths', '/accesspathslist.php', '/accesspathcreate.php', '/accesspathview.php', '/permissionassign.php'))) { ?> aria-current="page"<?php } ?>>
		<?php Translate("Access-Paths"); ?>
	</a>
	<?php } ?>

	<?php if (HasAccess(ACL_MOD_SETTINGS, ACL_ACTION_CHANGE)) { ?>
	<a class="gh-tab" href="settings"<?php if ($svnNavIs(array('/settings', '/settings.php'))) { ?> aria-current="page"<?php } ?>>
		<?php Translate("Settings"); ?>
	</a>
	<?php } ?>

	<?php
	// 建立/指派類動作（舊版是各 tab 的下拉子選單，這裡收斂成右側「Create」選單）
	$svnQuickActions = array();
	if (IsProviderActive(PROVIDER_REPOSITORY_EDIT) && HasAccess(ACL_MOD_REPO, ACL_ACTION_ADD)) {
		$svnQuickActions[] = array('href' => 'repositories/new', 'label' => 'Create repository');
	}
	if (IsProviderActive(PROVIDER_USER_EDIT) && HasAccess(ACL_MOD_USER, ACL_ACTION_ADD)) {
		$svnQuickActions[] = array('href' => 'users/new', 'label' => 'Create user');
	}
	if (IsProviderActive(PROVIDER_GROUP_EDIT) && HasAccess(ACL_MOD_GROUP, ACL_ACTION_ADD)) {
		$svnQuickActions[] = array('href' => 'groups/new', 'label' => 'Create group');
	}
	if (IsProviderActive(PROVIDER_GROUP_EDIT) && HasAccess(ACL_MOD_GROUP, ACL_ACTION_ASSIGN)) {
		$svnQuickActions[] = array('href' => 'users/group-assign', 'label' => 'Memberships');
	}
	if (IsProviderActive(PROVIDER_ACCESSPATH_EDIT) && HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_ADD)) {
		$svnQuickActions[] = array('href' => 'access-paths/new', 'label' => 'Create access-path');
	}
	if (IsProviderActive(PROVIDER_ACCESSPATH_EDIT) && HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_ASSIGN)) {
		$svnQuickActions[] = array('href' => 'permissions', 'label' => 'Permissions');
	}
	if (HasAccess(ACL_MOD_SETTINGS, ACL_ACTION_CHANGE)) {
		$svnQuickActions[] = array('href' => 'settings', 'label' => 'Backend');
	}
	?>
	<?php if (!empty($svnQuickActions)) { ?>
	<div class="gh-tab-spacer">
		<details class="gh-menu">
			<summary><?php Translate("Create"); ?> &#9662;</summary>
			<div class="gh-menu-panel">
				<?php foreach ($svnQuickActions as $svnAction) { ?>
				<a href="<?php print($svnAction['href']); ?>"><?php Translate($svnAction['label']); ?></a>
				<?php } ?>
			</div>
		</details>
	</div>
	<?php } ?>
</nav>
