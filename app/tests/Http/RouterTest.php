<?php
namespace SvnAdmin\Tests\Http;

use PHPUnit\Framework\TestCase;
use ReflectionProperty;
use SvnAdmin\Http\Request;
use SvnAdmin\Http\Response;
use SvnAdmin\Http\Router;

/**
 * Tests the router in isolation. The {@see Response} object intentionally has
 * no getters, so the tests read its state through reflection instead of
 * changing production code.
 */
final class RouterTest extends TestCase
{
	/** @var array<string,mixed> */
	private $serverBackup;

	protected function setUp(): void
	{
		$this->serverBackup = $_SERVER;
		$_SERVER['SCRIPT_NAME'] = '/svnadmin/index.php';
	}

	protected function tearDown(): void
	{
		$_SERVER = $this->serverBackup;
	}

	public function testGetRouteMatches(): void
	{
		$router = new Router();
		$router->get('/users', static function (Request $request): Response {
			return (new Response())->setBody('user list');
		});

		$response = $router->dispatch($this->request('/svnadmin/users'));

		$this->assertSame(200, $this->statusOf($response));
		$this->assertSame('user list', $this->bodyOf($response));
	}

	public function testUnmatchedPathReturns404(): void
	{
		$router = new Router();
		$router->get('/users', static function (): Response {
			return new Response();
		});

		$response = $router->dispatch($this->request('/svnadmin/does-not-exist'));

		$this->assertSame(404, $this->statusOf($response));
		$this->assertSame('404 Not Found', $this->bodyOf($response));
	}

	public function testLegacyPathRedirectsWith301(): void
	{
		$router = new Router();
		$router->legacy('/userlist.php', '/users');
		$router->get('/users', static function (): Response {
			return (new Response())->setBody('user list');
		});

		$response = $router->dispatch($this->request('/svnadmin/userlist.php?filter=a'));

		$this->assertSame(301, $this->statusOf($response));
		$this->assertSame('/users?filter=a', $this->headerOf($response, 'Location'));
	}

	public function testLegacyPathsAreRegisteredAsAliases(): void
	{
		// Current production routing keeps the old .php URLs as aliases (200),
		// so both spellings have to reach the same handler.
		$router = new Router();
		$handler = static function (): Response {
			return (new Response())->setBody('same page');
		};
		$router->any('/users', $handler);
		$router->any('/userlist.php', $handler);

		$new = $router->dispatch($this->request('/svnadmin/users'));
		$old = $router->dispatch($this->request('/svnadmin/userlist.php'));

		$this->assertSame(200, $this->statusOf($new));
		$this->assertSame(200, $this->statusOf($old));
		$this->assertSame($this->bodyOf($new), $this->bodyOf($old));
	}

	public function testBasePathIsStrippedFromUri(): void
	{
		$router = new Router();
		$router->get('/', static function (): Response {
			return (new Response())->setBody('home');
		});
		$router->get('/users', static function (): Response {
			return (new Response())->setBody('users');
		});

		$this->assertSame('home', $this->bodyOf($router->dispatch($this->request('/svnadmin/'))));
		$this->assertSame('users', $this->bodyOf($router->dispatch($this->request('/svnadmin/users'))));

		// Outside of the /svnadmin prefix the same relative routes apply.
		$_SERVER['SCRIPT_NAME'] = '/index.php';
		$this->assertSame('users', $this->bodyOf($router->dispatch($this->request('/users'))));
	}

	public function testMethodMustMatch(): void
	{
		$router = new Router();
		$router->get('/users', static function (): Response {
			return (new Response())->setBody('get only');
		});

		$response = $router->dispatch($this->request('/svnadmin/users', 'POST'));

		$this->assertSame(404, $this->statusOf($response));
	}

	private function request(string $uri, string $method = 'GET'): Request
	{
		$_SERVER['REQUEST_METHOD'] = $method;
		$_SERVER['REQUEST_URI'] = $uri;
		return new Request();
	}

	private function statusOf(Response $response): int
	{
		return (new ReflectionProperty(Response::class, 'statusCode'))->getValue($response);
	}

	private function bodyOf(Response $response): string
	{
		return (new ReflectionProperty(Response::class, 'body'))->getValue($response);
	}

	private function headerOf(Response $response, string $name): ?string
	{
		$headers = (new ReflectionProperty(Response::class, 'headers'))->getValue($response);
		return $headers[$name] ?? null;
	}
}
