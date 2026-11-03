<?php
namespace SvnAdmin\Acl;

/**
 * One role which can be assigned to users.
 */
class AclRole
{
    private $name = NULL;
    private $description = NULL;

    public function __construct($name, $description=NULL)
    {
      $this->name = $name;
      $this->description = $description;
    }

    public function getName()
    {
      return $this->name;
    }

    public function getDescription()
    {
      return $this->description;
    }
}
