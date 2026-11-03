<?php
namespace SvnAdmin\Acl;

/**
 * One module (ACL_MOD_*) which permissions can be granted for.
 */
class AclModule
{
  private $name = NULL;

  public function __construct($name)
  {
    $this->name = $name;
  }

  public function getName()
  {
    return $this->name;
  }
}
