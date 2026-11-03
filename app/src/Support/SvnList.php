<?php
namespace SvnAdmin\Support;

/**
 * Holds the result of a "svn ls" call.
 *
 * @author Manuel Freiholz, insaneFactory.com
 */
class SvnList
{
	public $path = "";
	public $entries = array();
	public $curEntry = null;
}
