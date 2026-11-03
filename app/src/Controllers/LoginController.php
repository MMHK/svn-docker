<?php
namespace SvnAdmin\Controllers;

class LoginController extends Controller
{
    public function handle(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;

        // Authentication disabled.
        if (!$appEngine->isAuthenticationActive()) {
            $appEngine->forward(PAGE_HOME, null, true);
            return $this->response;
        }

        // If the user is already logged in, redirect to index.
        if ($appEngine->checkUserAuthentication(false)) {
            $appEngine->forward(PAGE_HOME, null, true);
            return $this->response;
        }

        $appTR->loadModule("login");

        // Handle login action (POST).
        if (check_request_var("login")) {
            $appEngine->handleAction("login_authentication");
        }

        // View data.
        $logged_out = check_request_var("loggedout");
        $this->setValue("LoggedOut", $logged_out);
        $this->processTemplate("login.html.php");

        return $this->response;
    }
}
