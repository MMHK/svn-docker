<?php
namespace SvnAdmin\Tests\Config;

use PHPUnit\Framework\TestCase;
use SvnAdmin\Config\Config;
use SvnAdmin\Config\ConfigFactory;

/**
 * Tests {@see Config} / {@see ConfigFactory} without touching application data.
 */
final class ConfigTest extends TestCase
{
	/** @var string */
	private $file;

	/** @var string */
	private $savedFile;

	protected function setUp(): void
	{
		$this->file = tempnam(sys_get_temp_dir(), 'config');
		$this->savedFile = tempnam(sys_get_temp_dir(), 'config-saved');

		$content = <<<'INI'
[Common]
FirstStart=0
BackupFolder=./data/backup/

[Subversion]
SVNAuthFile=/var/svnlocal/svn-access-file
Debug=on
Off=off
One=1
Zero=0

[Engine:Providers]
UserViewProviderType=passwd
INI;
		file_put_contents($this->file, $content);
	}

	protected function tearDown(): void
	{
		foreach (array($this->file, $this->savedFile) as $path) {
			if (is_file($path)) {
				unlink($path);
			}
		}
	}

	public function testFactoryCreatesConfig(): void
	{
		$config = ConfigFactory::create($this->file);

		$this->assertInstanceOf(Config::class, $config);
		$this->assertSame($this->file, $config->getConfigPath());
	}

	public function testGetValueAndDefault(): void
	{
		$config = ConfigFactory::create($this->file);

		$this->assertSame('/var/svnlocal/svn-access-file', $config->getValue('Subversion', 'SVNAuthFile'));
		$this->assertSame('fallback', $config->getValue('Subversion', 'Unknown', 'fallback'));
		$this->assertNull($config->getValue('Unknown', 'Unknown'));
	}

	public function testGetValueAsBoolean(): void
	{
		$config = ConfigFactory::create($this->file);

		$this->assertTrue($config->getValueAsBoolean('Subversion', 'Debug'));
		$this->assertTrue($config->getValueAsBoolean('Subversion', 'One'));
		$this->assertFalse($config->getValueAsBoolean('Subversion', 'Off'));
		$this->assertFalse($config->getValueAsBoolean('Subversion', 'Zero'));
		$this->assertTrue($config->getValueAsBoolean('Subversion', 'Unknown', true));
	}

	public function testSectionAndKeyHelpers(): void
	{
		$config = ConfigFactory::create($this->file);

		$this->assertTrue($config->getSectionExists('Subversion'));
		$this->assertFalse($config->getSectionExists('Missing'));
		$this->assertTrue($config->getValueExists('Engine:Providers', 'UserViewProviderType'));
		$this->assertFalse($config->getValueExists('Engine:Providers', 'Missing'));
		$this->assertContains('Common', $config->getSections());
		$this->assertContains('SVNAuthFile', $config->getSectionKeys('Subversion'));
	}

	public function testSaveToFileRoundTrip(): void
	{
		$config = ConfigFactory::create($this->file);
		$config->setValue('Subversion', 'SVNAuthFile', '/tmp/changed-access-file');
		$config->setValue('Common', 'FirstStart', '1');

		$this->assertTrue($config->saveToFile($this->savedFile));

		$reloaded = ConfigFactory::create($this->savedFile);
		$this->assertSame('/tmp/changed-access-file', $reloaded->getValue('Subversion', 'SVNAuthFile'));
		$this->assertSame('1', $reloaded->getValue('Common', 'FirstStart'));
		$this->assertSame('passwd', $reloaded->getValue('Engine:Providers', 'UserViewProviderType'));

		$written = file_get_contents($this->savedFile);
		$this->assertStringContainsString('[Subversion]', $written);
		$this->assertStringContainsString('SVNAuthFile=/tmp/changed-access-file', $written);
	}

	public function testMissingFileThrows(): void
	{
		$this->expectException(\Exception::class);
		new Config($this->file . '.missing');
	}
}
