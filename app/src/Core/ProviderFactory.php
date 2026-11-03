<?php
namespace SvnAdmin\Core;

class ProviderFactory
{
    public static function initialize(Engine $engine, \SvnAdmin\View\Template $template): void
    {
        $cfg = $engine->getConfig();

        // This deployment uses: passwd + svnauthfile + svnclient + basic auth
        // Initialize all providers directly (no conditional checks needed)

        // User view/edit provider (passwd)
        $userProvider = \SvnAdmin\Providers\PasswdUserProvider::getInstance();
        $engine->setUserViewProvider($userProvider);
        $engine->setUserEditProvider($userProvider);

        // Group view/edit provider (svnauthfile)
        $groupProvider = \SvnAdmin\Providers\AuthFileGroupAndPathProvider::getInstance();
        $engine->setGroupViewProvider($groupProvider);
        $engine->setGroupEditProvider($groupProvider);

        // Access path view/edit provider (svnauthfile)
        $pathProvider = \SvnAdmin\Providers\AuthFileGroupAndPathProvider::getInstance();
        $engine->setAccessPathViewProvider($pathProvider);
        $engine->setAccessPathEditProvider($pathProvider);

        // Repository view/edit provider (svnclient)
        $repoView = \SvnAdmin\Providers\RepositoryViewProvider::getInstance();
        $repoEdit = \SvnAdmin\Providers\RepositoryEditProvider::getInstance();
        $engine->setRepositoryViewProvider($repoView);
        $engine->setRepositoryEditProvider($repoEdit);

        // Authentication (basic)
        \SvnAdmin\Http\Session::instance()->start();
        $authenticator = new \SvnAdmin\Providers\EngineBaseAuthenticator();
        $engine->setAuthenticator($authenticator);

        // ACL manager
        $aclManager = new \SvnAdmin\Acl\FSAclManager($cfg->getValue("ACLManager", "UserRoleAssignmentFile"));
        $engine->setAclManager($aclManager);
        $template->setAcl($engine);
    }
}
