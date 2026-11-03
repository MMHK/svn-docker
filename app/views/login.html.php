<?php
/**
 * 登入頁（GitHub 登入頁風格：居中卡片）。
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
	<base href="<?php print($svnBasePath); ?>/">
	<title>iF.SVNAdmin | Login</title>
	<meta name="Author" content="Manuel Freiholz, insaneFactory.com">
	<link rel="stylesheet" href="<?php print($svnBasePath); ?>/assets/app.css">
	<script src="<?php print($svnBasePath); ?>/assets/app.js" defer></script>
</head>
<body class="font-sans antialiased">

	<div class="login-shell">
		<div class="flex w-full max-w-sm items-center justify-end">
			<button type="button" class="btn px-2 py-1 text-xs" data-color-mode-toggle
				title="Color mode" aria-label="Color mode">
				<span data-color-mode-icon aria-hidden="true">&#9681;</span>
			</button>
		</div>

		<div class="login-card">
			<div class="login-logo">
				<img src="<?php print($svnBasePath); ?>/templates/images/logo.png" width="24" height="40" alt="iF.SVNAdmin">
				<span>iF.SVNAdmin</span>
			</div>

			<h1 class="mt-4 mb-1 text-center text-xl font-semibold"><?php Translate("Login"); ?></h1>
			<p class="mb-4 text-center text-sm text-fg-muted">
				<?php Translate("You must login to get access to the application."); ?>
			</p>

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

			<form action="login.php" method="POST">
				<div class="login-field">
					<label for="loginname"><?php Translate("Username"); ?></label>
					<input type="text" name="loginname" id="loginname" class="input" autocomplete="username" autofocus>
				</div>

				<div class="login-field">
					<label for="loginpass"><?php Translate("Password"); ?></label>
					<input type="password" name="loginpass" id="loginpass" class="input" autocomplete="current-password">
				</div>

				<button type="submit" name="login" value="<?php Translate("Login"); ?>" class="btn btn-primary mt-1 w-full justify-center">
					<?php Translate("Login"); ?>
				</button>
			</form>
		</div>

		<div class="flex flex-wrap items-center justify-center gap-3 text-xs text-fg-muted">
			<?php foreach (GetArrayValue("LocaleList") as $l): ?>
			<a class="flex items-center gap-1 hover:text-accent-fg" href="<?php PrintCurrentScriptName(); ?>?locale=<?php print($l->getLocale()); ?>" title="<?php Translate("Translated by"); ?> <?php print($l->getAuthor()); ?>">
				<img src="<?php print($svnBasePath); ?>/templates/flags/<?php print($l->getLocale()); ?>.png" alt="">
				<?php print($l->getName()); ?>
			</a>
			<?php endforeach; ?>
		</div>
	</div>

</body>
</html>
