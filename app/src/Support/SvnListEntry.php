<?php
namespace SvnAdmin\Support;

/**
 * One entry of a "svn ls" result.
 *
 * @author Manuel Freiholz, insaneFactory.com
 */
class SvnListEntry
{
	public $rev = 1;
	public $author = "";
	public $date = "";
	public $size = 0;
	public $name = "";
	public $isdir = false;
}
