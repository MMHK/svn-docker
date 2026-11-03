<?php
namespace SvnAdmin\Tests\View;

use PHPUnit\Framework\TestCase;
use SvnAdmin\View\Template;

/**
 * Pure logic tests for the historical template engine.
 */
final class TemplateTest extends TestCase
{
	public function testReplacesVariables(): void
	{
		$template = new Template();
		$this->assertTrue($template->loadFromString('Hello ${NAME}!'));
		$this->assertTrue($template->addReplacement('NAME', 'World'));

		$this->assertSame('Hello World!', $template->getProcessedTemplate());
	}

	public function testMissingVariableIsLeftUntouched(): void
	{
		$template = new Template();
		$template->loadFromString('A${MISSING}B');

		$this->assertSame('A${MISSING}B', $template->getProcessedTemplate());
	}

	public function testIfdefKeepsBlockWhenDefineIsSet(): void
	{
		$template = new Template();
		$template->loadFromString('A[{IFDEF FLAG}]B[{/IFDEF}]C');
		$template->addDefine('FLAG');

		$this->assertSame('ABC', $template->getProcessedTemplate());
	}

	public function testIfdefRemovesBlockWhenDefineIsMissing(): void
	{
		$template = new Template();
		$template->loadFromString('A[{IFDEF FLAG}]B[{/IFDEF}]C');

		$this->assertSame('AC', $template->getProcessedTemplate());
	}

	public function testIfdefSupportsNegation(): void
	{
		$template = new Template();
		$template->loadFromString('A[{IFDEF !FLAG}]B[{/IFDEF}]C');

		$this->assertSame('ABC', $template->getProcessedTemplate());
	}

	public function testLoopOverScalars(): void
	{
		$template = new Template();
		$template->loadFromString('Items:[{LOOP NAMES}]${NAMES}{$},[{/LOOP}]');
		$template->addReplacement('NAMES', array('one', 'two'));

		$this->assertSame('Items:one,two,', $template->getProcessedTemplate());
	}

	public function testLoopOverObjectsUsesGetters(): void
	{
		$template = new Template();
		$template->loadFromString('Users:[{LOOP USERS}]<${USERS}{name}>[{/LOOP}]');
		$template->addReplacement('USERS', array(
			new TemplateRow('alice'),
			new TemplateRow('bob'),
		));

		$this->assertSame('Users:<alice><bob>', $template->getProcessedTemplate());
	}

	public function testLoopOverMissingVariableIsRemoved(): void
	{
		$template = new Template();
		$template->loadFromString('A[{LOOP MISSING}]x[{/LOOP}]B');

		$this->assertSame('AB', $template->getProcessedTemplate());
	}
}

/**
 * Small fixture object: the template engine resolves `${...}{name}` through
 * the getName() getter.
 */
final class TemplateRow
{
	/** @var string */
	private $name;

	public function __construct(string $name)
	{
		$this->name = $name;
	}

	public function getName(): string
	{
		return $this->name;
	}
}
