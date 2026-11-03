<?php
/**
 * 全站外框：文件開頭、<head>、頂部 header 與內容區起點。
 * GitHub Primer 風格；深色模式以 <html data-color-mode> + cookie 控制。
 */
$svnBasePath = rtrim(str_replace('\\', '/', dirname($_SERVER['SCRIPT_NAME'] ?? '')), '/');
$svnColorMode = $_COOKIE['color_mode'] ?? 'auto';
if (!in_array($svnColorMode, array('auto', 'light', 'dark'), true)) {
	$svnColorMode = 'auto';
}
?><!DOCTYPE html>
<html data-color-mode="<?php print($svnColorMode); ?>">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<?php /* 修 R5：嵌套路由（/svnadmin/users/view）下，相對資源必須以應用根為準 */ ?>
	<base href="<?php print($svnBasePath); ?>/">
	<title>iF.SVNAdmin | <?php PrintApplicationVersion(); ?></title>
	<meta name="Author" content="Manuel Freiholz, insaneFactory.com">
	<link rel="stylesheet" href="<?php print($svnBasePath); ?>/assets/legacy.css">
	<link rel="stylesheet" href="<?php print($svnBasePath); ?>/assets/app.css">
	<script src="<?php print($svnBasePath); ?>/assets/app.js" defer></script>
</head>
<body class="min-h-screen bg-canvas-default font-sans text-fg-default antialiased">

	<header class="gh-topbar">
		<a class="gh-brand" href="./">
			<img src="<?php print($svnBasePath); ?>/templates/images/logo.png" width="24" height="40" alt="">
			<span>iF.SVNAdmin</span>
		</a>

		<button type="button" class="gh-tabs-toggle" data-nav-toggle="global-navigation"
			aria-controls="global-navigation" aria-expanded="false" aria-label="Menu">
			&#9776;
		</button>

		<div class="gh-topbar-controls">
			<label class="flex items-center gap-1 text-xs">
				<span class="sr-only"><?php Translate("Language"); ?></span>
				<select id="locale-selector" class="gh-select" aria-label="<?php Translate("Language"); ?>">
					<?php foreach (GetArrayValue("LocaleList") as $l) { ?>
					<option value="<?php print($l->locale); ?>"<?php if (CurrentLocale() == $l->locale) { ?> selected="selected"<?php } ?>>
						<?php print($l->name); ?>
					</option>
					<?php } ?>
				</select>
			</label>

			<button type="button" class="gh-icon-btn" data-color-mode-toggle
				title="Color mode" aria-label="Color mode">
				<span data-color-mode-icon aria-hidden="true">&#9681;</span>
			</button>

			<?php if (IsUserLoggedIn()) { ?>
			<details class="gh-menu">
				<summary>
					<?php Translate("Welcome"); ?> <b><?php PrintSessionUsername(); ?></b>
				</summary>
				<div class="gh-menu-panel">
					<?php if (IsProviderActive(PROVIDER_USER_EDIT) && HasAccess(ACL_MOD_USER, ACL_ACTION_CHANGEPASS)) { ?>
					<a href="users/change-password?username=<?php SessionUsername(); ?>"><?php Translate("Change password"); ?></a>
					<?php } ?>
					<a href="logout"><?php Translate("Logout"); ?></a>
				</div>
			</details>
			<?php } ?>
		</div>
	</header>

<?php include __DIR__ . '/global-navigation.php'; ?>

	<main class="gh-main">

		<?php if (HasAppExceptions()) : ?>
		<div class="flash flash-error">
			<strong><?php Translate("Exception list"); ?></strong>
			<ul class="mt-1 list-disc pl-6">
				<?php foreach (AppEngine()->getExceptions() as $item) : ?>
				<li><?php print($item->getMessage()); ?></li>
				<?php endforeach; ?>
			</ul>
		</div>
		<?php endif; ?>

		<?php if (HasAppMessages()) : ?>
		<div class="flash flash-success">
			<strong><?php Translate("Message list"); ?></strong>
			<ul class="mt-1 list-disc pl-6">
				<?php foreach (AppEngine()->getMessages() as $item) : ?>
				<li><?php print($item); ?></li>
				<?php endforeach; ?>
			</ul>
		</div>
		<?php endif; ?>
